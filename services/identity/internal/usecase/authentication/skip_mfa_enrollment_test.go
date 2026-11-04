package authentication

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestSkipMFAEnrollmentEntersConsent(t *testing.T) {
	t.Parallel()

	fixture := newSkipMFAEnrollmentFixture(t, oauthclientdomain.ConsentModeRequired, false)

	output, err := fixture.service.SkipMFAEnrollment(t.Context(), SkipMFAEnrollmentInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.Equal(t, transactiondomain.StatusPending, fixture.transaction.Status)
	require.Equal(t, transactiondomain.NextStepConsent, fixture.transaction.NextStep)
	require.Equal(t, authenticationdomain.LevelSingleFactor, *fixture.transaction.AuthenticationLevel)
	require.Same(t, fixture.transaction, fixture.transactions.updated)
}

func TestSkipMFAEnrollmentUsesTrustedClientConsent(t *testing.T) {
	t.Parallel()

	fixture := newSkipMFAEnrollmentFixture(t, oauthclientdomain.ConsentModeTrusted, false)

	output, err := fixture.service.SkipMFAEnrollment(t.Context(), SkipMFAEnrollmentInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.Equal(t, transactiondomain.StatusReady, fixture.transaction.Status)
	require.Equal(t, transactiondomain.NextStepReady, fixture.transaction.NextStep)
	require.Equal(t, fixture.transaction.RequestedScopes, fixture.transaction.GrantedScopes)
	require.Equal(t, transactiondomain.AuthorizationSourceTrustedClient, *fixture.transaction.AuthorizationSource)
}

func TestSkipMFAEnrollmentRejectsInvalidState(t *testing.T) {
	t.Parallel()

	fixture := newSkipMFAEnrollmentFixture(t, oauthclientdomain.ConsentModeRequired, false)
	fixture.transaction.NextStep = transactiondomain.NextStepMFA

	_, err := fixture.service.SkipMFAEnrollment(t.Context(), SkipMFAEnrollmentInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
	})

	require.True(t, apperr.IsCode(err, "mfa_enrollment_invalid_state"))
	require.Nil(t, fixture.transactions.updated)
}

func TestSkipMFAEnrollmentRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	fixture := newSkipMFAEnrollmentFixture(t, oauthclientdomain.ConsentModeRequired, false)

	_, err := fixture.service.SkipMFAEnrollment(t.Context(), SkipMFAEnrollmentInput{})

	require.True(t, apperr.IsCode(err, "mfa_enrollment_invalid_input"))
	require.Zero(t, fixture.tx.calls)
}

type skipMFAEnrollmentFixture struct {
	service      *Service
	transaction  *transactiondomain.AuthorizationTransaction
	transactions *skipMFAEnrollmentTransactionRepository
	tx           *skipMFAEnrollmentTransactionManager
}

func newSkipMFAEnrollmentFixture(
	t *testing.T,
	consentMode oauthclientdomain.ConsentMode,
	rememberedGrantCovers bool,
) skipMFAEnrollmentFixture {
	t.Helper()

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	identityID := identitydomain.ID(uuid.New())
	transaction := &transactiondomain.AuthorizationTransaction{
		ID:                   transactiondomain.ID(uuid.New()),
		OAuthClientID:        oauthclientdomain.ID(uuid.New()),
		RequestedScopes:      []oauthdomain.Scope{"openid", "profile"},
		Status:               transactiondomain.StatusPending,
		NextStep:             transactiondomain.NextStepMFAEnrollment,
		IdentityID:           &identityID,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword},
		ExpiresAt:            now.Add(10 * time.Minute),
	}
	facts, err := flowdomain.NewFacts(nil, consentMode, rememberedGrantCovers)
	require.NoError(t, err)
	transactions := &skipMFAEnrollmentTransactionRepository{transaction: transaction}
	tx := &skipMFAEnrollmentTransactionManager{}
	service := &Service{
		authorizationTransactions: transactions,
		authenticationFlowFacts:   &factsReaderStub{facts: facts},
		tx:                        tx,
		clock:                     skipMFAEnrollmentClock{now: now},
	}

	return skipMFAEnrollmentFixture{
		service:      service,
		transaction:  transaction,
		transactions: transactions,
		tx:           tx,
	}
}

type skipMFAEnrollmentTransactionRepository struct {
	transaction *transactiondomain.AuthorizationTransaction
	updated     *transactiondomain.AuthorizationTransaction
}

func (r *skipMFAEnrollmentTransactionRepository) Get(
	context.Context,
	transactiondomain.ID,
) (*transactiondomain.AuthorizationTransaction, error) {
	if r.transaction == nil {
		return nil, transactiondomain.ErrNotFound
	}
	return r.transaction, nil
}

func (r *skipMFAEnrollmentTransactionRepository) Update(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
) error {
	r.updated = transaction
	return nil
}

type skipMFAEnrollmentTransactionManager struct {
	calls int
}

func (m *skipMFAEnrollmentTransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type skipMFAEnrollmentClock struct {
	now time.Time
}

func (c skipMFAEnrollmentClock) Now() time.Time {
	return c.now
}
