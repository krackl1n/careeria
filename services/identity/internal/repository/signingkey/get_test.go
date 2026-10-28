package signingkey

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
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

func TestGetActiveSelectsUsableKeyByAlgorithm(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	activatedAt := now.Add(-time.Hour)
	want := row{
		ID:            uuid.New(),
		KeyID:         "key-1",
		Algorithm:     "EdDSA",
		PublicJWK:     []byte(`{"kty":"OKP"}`),
		PrivateKeyRef: "vault://identity/key-1",
		Status:        "active",
		CreatedAt:     now.Add(-2 * time.Hour),
		NotBefore:     now.Add(-2 * time.Hour),
		ActivatedAt:   &activatedAt,
	}
	executor := &executorStub{row: &dataRowStub{value: want}}
	repository := NewPostgresRepository(executorProviderStub{executor: executor})

	actual, err := repository.GetActive(t.Context(), signingkeydomain.AlgorithmEdDSA, now)
	require.NoError(t, err)
	require.Equal(t, signingkeydomain.ID(want.ID), actual.ID)
	require.Equal(t, signingkeydomain.KeyID(want.KeyID), actual.KeyID)
	require.Equal(t, signingkeydomain.AlgorithmEdDSA, actual.Algorithm)
	require.Equal(t, signingkeydomain.StatusActive, actual.Status)
	require.Len(t, executor.args, 2)
	require.Equal(t, signingkeydomain.AlgorithmEdDSA, executor.args[0])
	require.Equal(t, now, executor.args[1])
	require.True(t, strings.Contains(executor.query, "status = 'active'"))
	require.True(t, strings.Contains(executor.query, "not_before <= $2"))
	require.True(t, strings.Contains(executor.query, "expires_at IS NULL OR expires_at > $2"))
}

func TestGetActiveReturnsDomainNotFound(t *testing.T) {
	t.Parallel()

	executor := &executorStub{row: errorRowStub{err: pgx.ErrNoRows}}
	repository := NewPostgresRepository(executorProviderStub{executor: executor})

	_, err := repository.GetActive(
		t.Context(),
		signingkeydomain.AlgorithmEdDSA,
		time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC),
	)
	require.ErrorIs(t, err, signingkeydomain.ErrNotFound)
}

func TestGetActiveWrapsDatabaseError(t *testing.T) {
	t.Parallel()

	databaseErr := errors.New("database unavailable")
	executor := &executorStub{row: errorRowStub{err: databaseErr}}
	repository := NewPostgresRepository(executorProviderStub{executor: executor})

	_, err := repository.GetActive(
		t.Context(),
		signingkeydomain.AlgorithmEdDSA,
		time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC),
	)
	require.ErrorIs(t, err, databaseErr)
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

type dataRowStub struct{ value row }

func (source *dataRowStub) Scan(destinations ...any) error {
	*(destinations[0].(*uuid.UUID)) = source.value.ID
	*(destinations[1].(*string)) = source.value.KeyID
	*(destinations[2].(*string)) = source.value.Algorithm
	*(destinations[3].(*[]byte)) = append([]byte(nil), source.value.PublicJWK...)
	*(destinations[4].(*string)) = source.value.PrivateKeyRef
	*(destinations[5].(*string)) = source.value.Status
	*(destinations[6].(*time.Time)) = source.value.CreatedAt
	*(destinations[7].(*time.Time)) = source.value.NotBefore
	*(destinations[8].(**time.Time)) = source.value.ActivatedAt
	*(destinations[9].(**time.Time)) = source.value.RetiringAt
	*(destinations[10].(**time.Time)) = source.value.RetiredAt
	*(destinations[11].(**time.Time)) = source.value.ExpiresAt
	*(destinations[12].(**time.Time)) = source.value.CompromisedAt
	return nil
}
