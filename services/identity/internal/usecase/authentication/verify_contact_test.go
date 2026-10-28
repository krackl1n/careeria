package authentication

import (
	"context"
	"errors"
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
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
	"github.com/stretchr/testify/require"
)

func TestVerifyEmailActivatesIdentityAndEntersMFAEnrollment(t *testing.T) {
	t.Parallel()

	fixture := newEmailVerificationFixture(t)

	output, err := fixture.service.VerifyEmail(t.Context(), VerifyEmailInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "000000",
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.Nil(t, fixture.transaction.Verification)
	require.Equal(t, transactiondomain.NextStepMFAEnrollment, fixture.transaction.NextStep)
	require.Equal(t, identitydomain.StatusActive, fixture.identity.Status)
	require.NotNil(t, fixture.identity.Emails[0].VerifiedAt)
	require.Same(t, fixture.identity, fixture.identities.updated)
	require.Same(t, fixture.transaction, fixture.transactions.updated)
	require.Equal(t, 1, fixture.verifications.verifyEmailCalls)
	require.Equal(t, "000000", fixture.verifications.code)
}

func TestVerifyPhoneActivatesIdentityAndEntersMFAEnrollment(t *testing.T) {
	t.Parallel()

	fixture := newPhoneVerificationFixture(t)

	output, err := fixture.service.VerifyPhone(t.Context(), VerifyPhoneInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "000000",
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.Nil(t, fixture.transaction.Verification)
	require.Equal(t, transactiondomain.NextStepMFAEnrollment, fixture.transaction.NextStep)
	require.Equal(t, identitydomain.StatusActive, fixture.identity.Status)
	require.NotNil(t, fixture.identity.Phones[0].VerifiedAt)
	require.Same(t, fixture.identity, fixture.identities.updated)
	require.Equal(t, 1, fixture.verifications.verifyPhoneCalls)
}

func TestVerifyEmailReturnsInternalVerificationError(t *testing.T) {
	t.Parallel()

	fixture := newEmailVerificationFixture(t)
	fixture.verifications.verifyEmailErr = errors.New("storage failed")

	_, err := fixture.service.VerifyEmail(t.Context(), VerifyEmailInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "111111",
	})

	require.ErrorContains(t, err, "verify email challenge")
	require.Error(t, fixture.tx.callbackErr)
	require.Equal(t, 1, fixture.verifications.verifyEmailCalls)
	require.Nil(t, fixture.identities.updated)
	require.Nil(t, fixture.transactions.updated)
	require.Equal(t, identitydomain.StatusPending, fixture.identity.Status)
	require.Nil(t, fixture.identity.Emails[0].VerifiedAt)
}

func TestVerifyPhoneMapsUnavailableChallenge(t *testing.T) {
	t.Parallel()

	fixture := newPhoneVerificationFixture(t)
	fixture.verifications.verifyPhoneErr = contactverificationservice.ErrChallengeUnavailable

	_, err := fixture.service.VerifyPhone(t.Context(), VerifyPhoneInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "000000",
	})

	require.True(t, apperr.IsType(err, apperr.TypeConflict))
	require.True(t, apperr.IsCode(err, "contact_verification_invalid_state"))
	require.Error(t, fixture.tx.callbackErr)
	require.Nil(t, fixture.identities.updated)
	require.Nil(t, fixture.transactions.updated)
}

func TestVerifyEmailRejectsInvalidTransactionState(t *testing.T) {
	t.Parallel()

	fixture := newEmailVerificationFixture(t)
	fixture.transaction.NextStep = transactiondomain.NextStepConsent

	_, err := fixture.service.VerifyEmail(t.Context(), VerifyEmailInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "000000",
	})

	require.True(t, apperr.IsCode(err, "contact_verification_invalid_state"))
	require.Equal(t, 1, fixture.verifications.verifyEmailCalls)
}

func TestVerifyContactRejectsInvalidInputBeforeTransaction(t *testing.T) {
	t.Parallel()

	fixture := newEmailVerificationFixture(t)

	_, err := fixture.service.VerifyEmail(t.Context(), VerifyEmailInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Code:                       "12345a",
	})

	require.True(t, apperr.IsCode(err, "contact_verification_invalid_input"))
	require.Zero(t, fixture.tx.calls)
}

type verificationFixture struct {
	service       *Service
	transaction   *transactiondomain.AuthorizationTransaction
	identity      *identitydomain.Identity
	transactions  *verificationTransactionRepository
	identities    *verificationIdentityRepository
	verifications *verificationServiceStub
	tx            *verificationTransactionManager
}

func newEmailVerificationFixture(t *testing.T) verificationFixture {
	t.Helper()

	now := verificationTestTime()
	identity, err := identitydomain.NewWithEmail(
		identitydomain.ID(uuid.New()),
		identitydomain.EmailID(uuid.New()),
		identitydomain.EmailAddress("user@example.com"),
		now.Add(-2*time.Minute),
	)
	require.NoError(t, err)

	return newVerificationFixture(t, identity, transactiondomain.NextStepEmailVerification, transactiondomain.VerificationChannelEmail, identity.Emails[0].ID.UUID())
}

func newPhoneVerificationFixture(t *testing.T) verificationFixture {
	t.Helper()

	now := verificationTestTime()
	identity, err := identitydomain.NewWithPhone(
		identitydomain.ID(uuid.New()),
		identitydomain.PhoneID(uuid.New()),
		identitydomain.PhoneNumber("+79991234567"),
		now.Add(-2*time.Minute),
	)
	require.NoError(t, err)

	return newVerificationFixture(t, identity, transactiondomain.NextStepPhoneVerification, transactiondomain.VerificationChannelPhone, identity.Phones[0].ID.UUID())
}

func newVerificationFixture(
	t *testing.T,
	identity *identitydomain.Identity,
	step transactiondomain.NextStep,
	channel transactiondomain.VerificationChannel,
	contactID uuid.UUID,
) verificationFixture {
	t.Helper()

	now := verificationTestTime()
	identityID := identity.ID
	reference, err := transactiondomain.NewVerificationReference(channel, contactID, uuid.New(), now.Add(5*time.Minute))
	require.NoError(t, err)
	transaction := &transactiondomain.AuthorizationTransaction{
		ID:                   transactiondomain.ID(uuid.New()),
		OAuthClientID:        oauthclientdomain.ID(uuid.New()),
		RequestedScopes:      []oauthdomain.Scope{"openid", "profile"},
		Status:               transactiondomain.StatusPending,
		NextStep:             step,
		IdentityID:           &identityID,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword},
		Verification:         &reference,
		ExpiresAt:            now.Add(10 * time.Minute),
	}
	facts, err := flowdomain.NewFacts(nil, oauthclientdomain.ConsentModeRequired, false)
	require.NoError(t, err)
	transactions := &verificationTransactionRepository{transaction: transaction}
	identities := &verificationIdentityRepository{identity: identity}
	verifications := &verificationServiceStub{}
	tx := &verificationTransactionManager{}
	service := &Service{
		authorizationTransactions: transactions,
		identities:                identities,
		authenticationFlowFacts:   &factsReaderStub{facts: facts},
		contactVerifications:      verifications,
		tx:                        tx,
		clock:                     verificationClock{now: now},
	}

	return verificationFixture{
		service:       service,
		transaction:   transaction,
		identity:      identity,
		transactions:  transactions,
		identities:    identities,
		verifications: verifications,
		tx:            tx,
	}
}

type verificationTransactionRepository struct {
	transaction *transactiondomain.AuthorizationTransaction
	updated     *transactiondomain.AuthorizationTransaction
}

func (r *verificationTransactionRepository) Get(context.Context, transactiondomain.ID) (*transactiondomain.AuthorizationTransaction, error) {
	if r.transaction == nil {
		return nil, transactiondomain.ErrNotFound
	}
	return r.transaction, nil
}

func (r *verificationTransactionRepository) Update(_ context.Context, transaction *transactiondomain.AuthorizationTransaction) error {
	r.updated = transaction
	return nil
}

type verificationIdentityRepository struct {
	identity *identitydomain.Identity
	updated  *identitydomain.Identity
}

func (*verificationIdentityRepository) Create(context.Context, *identitydomain.Identity) error {
	return nil
}

func (r *verificationIdentityRepository) Get(context.Context, identitydomain.ID) (*identitydomain.Identity, error) {
	if r.identity == nil {
		return nil, identitydomain.ErrNotFound
	}
	return r.identity, nil
}

func (*verificationIdentityRepository) GetByEmail(context.Context, identitydomain.EmailAddress) (*identitydomain.Identity, error) {
	return nil, identitydomain.ErrNotFound
}

func (*verificationIdentityRepository) GetByPhone(context.Context, identitydomain.PhoneNumber) (*identitydomain.Identity, error) {
	return nil, identitydomain.ErrNotFound
}

func (r *verificationIdentityRepository) Update(_ context.Context, identity *identitydomain.Identity) error {
	r.updated = identity
	return nil
}

type verificationServiceStub struct {
	verifyEmailCalls int
	verifyPhoneCalls int
	verifyEmailErr   error
	verifyPhoneErr   error
	code             string
}

func (*verificationServiceStub) StartEmail(
	context.Context,
	*transactiondomain.AuthorizationTransaction,
	identitydomain.Email,
	*sessiondomain.IPAddress,
	time.Time,
) error {
	return errors.New("unexpected email verification start")
}

func (*verificationServiceStub) StartPhone(
	context.Context,
	*transactiondomain.AuthorizationTransaction,
	identitydomain.Phone,
	*sessiondomain.IPAddress,
	time.Time,
) error {
	return errors.New("unexpected phone verification start")
}

func (s *verificationServiceStub) VerifyEmail(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	code string,
	_ *sessiondomain.IPAddress,
	_ time.Time,
) (identitydomain.EmailID, error) {
	s.verifyEmailCalls++
	s.code = code
	if s.verifyEmailErr != nil {
		return identitydomain.EmailID{}, s.verifyEmailErr
	}
	if transaction.NextStep != transactiondomain.NextStepEmailVerification ||
		transaction.Verification == nil ||
		transaction.Verification.Channel != transactiondomain.VerificationChannelEmail {
		return identitydomain.EmailID{}, contactverificationservice.ErrChallengeUnavailable
	}
	return identitydomain.EmailID(transaction.Verification.ContactID), nil
}

func (s *verificationServiceStub) VerifyPhone(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	code string,
	_ *sessiondomain.IPAddress,
	_ time.Time,
) (identitydomain.PhoneID, error) {
	s.verifyPhoneCalls++
	s.code = code
	if s.verifyPhoneErr != nil {
		return identitydomain.PhoneID{}, s.verifyPhoneErr
	}
	if transaction.NextStep != transactiondomain.NextStepPhoneVerification ||
		transaction.Verification == nil ||
		transaction.Verification.Channel != transactiondomain.VerificationChannelPhone {
		return identitydomain.PhoneID{}, contactverificationservice.ErrChallengeUnavailable
	}
	return identitydomain.PhoneID(transaction.Verification.ContactID), nil
}

type verificationTransactionManager struct {
	calls       int
	callbackErr error
}

func (m *verificationTransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	m.calls++
	m.callbackErr = fn(ctx)
	return m.callbackErr
}

type verificationClock struct {
	now time.Time
}

func (c verificationClock) Now() time.Time {
	return c.now
}

func verificationTestTime() time.Time {
	return time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
}
