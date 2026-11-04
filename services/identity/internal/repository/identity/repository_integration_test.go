package identity

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepositoryRoundTrip(t *testing.T) {
	dsn := os.Getenv("IDENTITY_TEST_DSN")
	if dsn == "" {
		t.Skip("IDENTITY_TEST_DSN is not set")
	}

	ctx := context.Background()
	client := newTestPostgresClient(t, ctx, dsn)
	defer client.Close()

	repository := NewPostgresRepository(client)
	now := time.Now().UTC().Truncate(time.Microsecond)
	address, err := identitydomain.NewEmailAddress("repository-" + uuid.NewString() + "@example.com")
	require.NoError(t, err)

	entity, err := identitydomain.NewWithEmail(
		identitydomain.ID(uuid.New()),
		identitydomain.EmailID(uuid.New()),
		address,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, repository.Create(ctx, entity))

	loaded, err := repository.Get(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ID, loaded.ID)
	require.Len(t, loaded.Emails, 1)

	require.NoError(t, loaded.VerifyEmail(loaded.Emails[0].ID, now.Add(time.Minute)))
	require.NoError(t, loaded.Activate(now.Add(2*time.Minute)))
	number, err := identitydomain.NewPhoneNumber(
		fmt.Sprintf("+79%09d", time.Now().UnixNano()%1_000_000_000),
	)
	require.NoError(t, err)
	require.NoError(t, loaded.AddPhone(
		identitydomain.PhoneID(uuid.New()),
		number,
		now.Add(3*time.Minute),
	))
	require.NoError(t, loaded.VerifyPhone(loaded.Phones[0].ID, now.Add(4*time.Minute)))
	provider, err := identitydomain.NewExternalProvider("oidc")
	require.NoError(t, err)
	subject, err := identitydomain.NewExternalSubject(uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, loaded.LinkExternalAccount(
		identitydomain.ExternalAccountID(uuid.New()),
		provider,
		subject,
		now.Add(5*time.Minute),
	))
	require.NoError(t, repository.Update(ctx, loaded))

	updated, err := repository.Get(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, identitydomain.StatusActive, updated.Status)
	require.Len(t, updated.Emails, 1)
	require.NotNil(t, updated.Emails[0].VerifiedAt)
	require.Len(t, updated.Phones, 1)
	require.Len(t, updated.ExternalAccounts, 1)

	duplicate, err := identitydomain.NewWithEmail(
		identitydomain.ID(uuid.New()),
		identitydomain.EmailID(uuid.New()),
		address,
		now,
	)
	require.NoError(t, err)
	require.Equal(t, identitydomain.ErrEmailAlreadyExists, repository.Create(ctx, duplicate))
	_, err = repository.Get(ctx, duplicate.ID)
	require.Equal(t, identitydomain.ErrNotFound, err)

	require.NoError(t, updated.Delete(now.Add(6*time.Minute)))
	require.NoError(t, repository.Delete(ctx, updated))
	_, err = repository.Get(ctx, updated.ID)
	require.Equal(t, identitydomain.ErrNotFound, err)
}

func newTestPostgresClient(t *testing.T, ctx context.Context, dsn string) *sharedpostgres.Client {
	t.Helper()

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	client, err := sharedpostgres.New(ctx, sharedpostgres.Config{
		Connection: sharedpostgres.ConnectionConfig{
			Host:     parsed.Hostname(),
			Port:     port,
			Database: parsed.Path[1:],
			User:     parsed.User.Username(),
			Password: password,
			SSLMode:  parsed.Query().Get("sslmode"),
		},
		MaxConns:          2,
		MinConns:          0,
		MaxConnLifetime:   time.Minute,
		MaxConnIdleTime:   time.Minute,
		HealthCheckPeriod: time.Minute,
	})
	require.NoError(t, err)

	return client
}
