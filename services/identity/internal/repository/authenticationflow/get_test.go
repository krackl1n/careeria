package authenticationflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestGetReturnsMinimalAuthenticationFlowFacts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	executor := &executorStub{row: &factsRowStub{
		methods: []string{"recovery_code", "totp"},
		mode:    "remember",
		covers:  true,
	}}
	reader := NewPostgresReader(executorProviderStub{executor: executor})
	identityID := identitydomain.ID(uuid.New())
	clientID := oauthclientdomain.ID(uuid.New())

	facts, err := reader.Get(
		t.Context(),
		identityID,
		clientID,
		[]oauthdomain.Scope{"openid", "profile"},
		now,
	)
	require.NoError(t, err)
	require.Equal(t, []authenticationdomain.Method{
		authenticationdomain.MethodRecoveryCode,
		authenticationdomain.MethodTOTP,
	}, facts.AvailableMFAMethods())
	require.True(t, facts.RememberedGrantCovers())
	require.Equal(t, oauthclientdomain.ConsentModeRemember, facts.ConsentMode())
	require.Equal(t, []any{
		identityID.UUID(),
		clientID.UUID(),
		[]string{"openid", "profile"},
		now,
	}, executor.args)

	query := strings.ToLower(executor.query)
	require.Contains(t, query, "confirmed_at is not null")
	require.Contains(t, query, "disabled_at is null")
	require.NotContains(t, query, "secret_encrypted")
	require.NotContains(t, query, "public_key")
	require.NotContains(t, query, "credential_id")
}

func TestGetReturnsEmptyMFAAvailability(t *testing.T) {
	t.Parallel()

	executor := &executorStub{row: &factsRowStub{mode: "required"}}
	reader := NewPostgresReader(executorProviderStub{executor: executor})
	facts, err := reader.Get(
		t.Context(),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		[]oauthdomain.Scope{"openid"},
		time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.False(t, facts.HasUsableMFA())
}

func TestGetMapsMissingClientToFactsNotFound(t *testing.T) {
	t.Parallel()

	reader := NewPostgresReader(executorProviderStub{executor: &executorStub{row: errorRowStub{err: pgx.ErrNoRows}}})
	_, err := reader.Get(
		t.Context(),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		[]oauthdomain.Scope{"openid"},
		time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
	)
	require.ErrorIs(t, err, flowdomain.ErrFactsNotFound)
}

func TestGetValidatesQueryBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	executor := &executorStub{row: errorRowStub{err: errors.New("must not query")}}
	reader := NewPostgresReader(executorProviderStub{executor: executor})
	_, err := reader.Get(
		t.Context(),
		identitydomain.ID{},
		oauthclientdomain.ID(uuid.New()),
		[]oauthdomain.Scope{"openid"},
		time.Now(),
	)
	require.Error(t, err)
	require.Empty(t, executor.query)
}

type executorProviderStub struct{ executor sharedpostgres.Executor }

func (provider executorProviderStub) Executor(context.Context) sharedpostgres.Executor {
	return provider.executor
}

type executorStub struct {
	row   pgx.Row
	query string
	args  []any
}

func (executor *executorStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	executor.query = query
	executor.args = args
	return executor.row
}

func (*executorStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec call")
}

func (*executorStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query call")
}

func (*executorStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	panic("unexpected SendBatch call")
}

type errorRowStub struct{ err error }

func (row errorRowStub) Scan(...any) error { return row.err }

type factsRowStub struct {
	methods []string
	mode    string
	covers  bool
}

func (row *factsRowStub) Scan(destinations ...any) error {
	*(destinations[0].(*[]string)) = append([]string(nil), row.methods...)
	*(destinations[1].(*string)) = row.mode
	*(destinations[2].(*bool)) = row.covers
	return nil
}
