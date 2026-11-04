package authentication

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	"github.com/stretchr/testify/require"
)

func TestRegisterIdentityWithEmail(t *testing.T) {
	t.Parallel()

	fixture := newRegistrationFixture()
	email := "User@Example.com"
	middleName := "Петрович"

	output, err := fixture.service.RegisterIdentity(t.Context(), RegisterIdentityInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Email:                      &email,
		Password:                   "correct horse battery staple",
		Profile: &RegistrationProfileInput{
			FamilyName: "Иванов",
			GivenName:  "Иван",
			MiddleName: &middleName,
		},
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.NotNil(t, fixture.identities.created)
	require.Equal(t, identitydomain.StatusPending, fixture.identities.created.Status)
	require.Len(t, fixture.identities.created.Emails, 1)
	require.Equal(t, identitydomain.EmailAddress("user@example.com"), fixture.identities.created.Emails[0].Address)
	require.Equal(t, fixture.identities.created.ID, fixture.profiles.created.IdentityID)
	require.Equal(t, "Иванов", fixture.profiles.created.FamilyName)
	require.Equal(t, identityprofiledomain.EventTypeUpdated, fixture.outbox.created.EventType)
	require.Equal(t, identityprofiledomain.AggregateType, fixture.outbox.created.AggregateType)
	var profileEvent identityprofiledomain.UpdatedEvent
	require.NoError(t, json.Unmarshal(fixture.outbox.created.Payload, &profileEvent))
	require.Equal(t, fixture.profiles.created.IdentityID.UUID().String(), profileEvent.IdentitySub)
	require.Equal(t, passworddomain.Hash("encoded-password"), fixture.passwords.created.Credential.Hash)
	require.Equal(t, fixture.identities.created.ID, fixture.passwords.created.IdentityID)
	require.Equal(t, 1, fixture.verifications.emailCalls)
	require.Equal(t, fixture.identities.created.Emails[0].ID, fixture.verifications.emailID)
	require.Equal(t, identitydomain.EmailAddress("user@example.com"), fixture.verifications.emailAddress)
	require.Equal(t, transactiondomain.NextStepEmailVerification, fixture.transaction.NextStep)
	require.Equal(t, fixture.identities.created.ID, *fixture.transaction.IdentityID)
	require.NotNil(t, fixture.transaction.Verification)
	require.Equal(t, fixture.verifications.challengeID, fixture.transaction.Verification.ChallengeID)
	require.Same(t, fixture.transaction, fixture.transactions.updated)
}

func TestRegisterIdentityWithPhone(t *testing.T) {
	t.Parallel()

	fixture := newRegistrationFixture()
	phone := "+79991234567"

	output, err := fixture.service.RegisterIdentity(t.Context(), RegisterIdentityInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		PhoneNumber:                &phone,
		Password:                   "correct horse battery staple",
		Profile: &RegistrationProfileInput{
			FamilyName: "Иванов",
			GivenName:  "Иван",
		},
	})

	require.NoError(t, err)
	require.Same(t, fixture.transaction, output.AuthorizationTransaction)
	require.Len(t, fixture.identities.created.Phones, 1)
	require.Equal(t, identitydomain.PhoneNumber(phone), fixture.identities.created.Phones[0].Number)
	require.Equal(t, 1, fixture.verifications.phoneCalls)
	require.Equal(t, fixture.identities.created.Phones[0].ID, fixture.verifications.phoneID)
	require.Equal(t, identitydomain.PhoneNumber(phone), fixture.verifications.phoneNumber)
	require.Equal(t, transactiondomain.NextStepPhoneVerification, fixture.transaction.NextStep)
	require.Equal(t, transactiondomain.VerificationChannelPhone, fixture.transaction.Verification.Channel)
}

func TestRegisterIdentityRejectsInvalidInputBeforeTransaction(t *testing.T) {
	t.Parallel()

	fixture := newRegistrationFixture()
	email := "user@example.com"
	phone := "+79991234567"

	_, err := fixture.service.RegisterIdentity(t.Context(), RegisterIdentityInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Email:                      &email,
		PhoneNumber:                &phone,
		Password:                   "short",
		Profile:                    &RegistrationProfileInput{FamilyName: "Иванов", GivenName: "Иван"},
	})

	require.True(t, apperr.IsCode(err, "registration_invalid_input"))
	require.Zero(t, fixture.tx.calls)
	require.Nil(t, fixture.identities.created)
}

func TestRegisterIdentityMapsContactConflict(t *testing.T) {
	t.Parallel()

	fixture := newRegistrationFixture()
	fixture.identities.createErr = identitydomain.ErrEmailAlreadyExists
	email := "user@example.com"

	_, err := fixture.service.RegisterIdentity(t.Context(), RegisterIdentityInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Email:                      &email,
		Password:                   "correct horse battery staple",
		Profile:                    &RegistrationProfileInput{FamilyName: "Иванов", GivenName: "Иван"},
	})

	require.True(t, apperr.IsType(err, apperr.TypeConflict))
	require.True(t, apperr.IsCode(err, "registration_contact_already_exists"))
	require.Nil(t, fixture.profiles.created)
	require.Nil(t, fixture.passwords.created)
	require.Zero(t, fixture.verifications.emailCalls)
}

func TestRegisterIdentityRejectsAuthorizationTransactionState(t *testing.T) {
	t.Parallel()

	fixture := newRegistrationFixture()
	fixture.transaction.NextStep = transactiondomain.NextStepConsent
	email := "user@example.com"

	_, err := fixture.service.RegisterIdentity(t.Context(), RegisterIdentityInput{
		AuthorizationTransactionID: fixture.transaction.ID.UUID(),
		Email:                      &email,
		Password:                   "correct horse battery staple",
		Profile:                    &RegistrationProfileInput{FamilyName: "Иванов", GivenName: "Иван"},
	})

	require.True(t, apperr.IsCode(err, "registration_invalid_state"))
	require.Nil(t, fixture.identities.created)
}

type registrationFixture struct {
	service       *Service
	transaction   *transactiondomain.AuthorizationTransaction
	transactions  *registrationTransactionRepository
	identities    *registrationIdentityRepository
	profiles      *registrationProfileRepository
	outbox        *registrationOutbox
	passwords     *registrationPasswordRepository
	verifications *registrationVerificationIssuer
	tx            *registrationTransactionManager
}

func newRegistrationFixture() registrationFixture {
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	transaction := &transactiondomain.AuthorizationTransaction{
		ID:        transactiondomain.ID(uuid.New()),
		Status:    transactiondomain.StatusPending,
		NextStep:  transactiondomain.NextStepChooseAuthenticationMethod,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	transactions := &registrationTransactionRepository{transaction: transaction}
	identities := &registrationIdentityRepository{}
	profiles := &registrationProfileRepository{}
	outbox := &registrationOutbox{}
	passwords := &registrationPasswordRepository{}
	verifications := &registrationVerificationIssuer{
		challengeID: uuid.New(),
		expiresAt:   now.Add(5 * time.Minute),
	}
	tx := &registrationTransactionManager{}
	ids := &registrationIDGenerator{values: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	service := &Service{
		authorizationTransactions: transactions,
		identities:                identities,
		identityProfiles:          profiles,
		outbox:                    outbox,
		events:                    eventdomain.NewFactory(ids),
		passwords:                 passwords,
		contactVerifications:      verifications,
		passwordHasher:            registrationPasswordHasher{},
		tx:                        tx,
		ids:                       ids,
		clock:                     registrationClock{now: now},
	}

	return registrationFixture{
		service:       service,
		transaction:   transaction,
		transactions:  transactions,
		identities:    identities,
		profiles:      profiles,
		outbox:        outbox,
		passwords:     passwords,
		verifications: verifications,
		tx:            tx,
	}
}

type registrationOutbox struct{ created eventdomain.Message }

func (r *registrationOutbox) Create(_ context.Context, message eventdomain.Message) error {
	r.created = message
	return nil
}

type registrationTransactionRepository struct {
	transaction *transactiondomain.AuthorizationTransaction
	updated     *transactiondomain.AuthorizationTransaction
}

func (r *registrationTransactionRepository) Get(
	_ context.Context,
	_ transactiondomain.ID,
) (*transactiondomain.AuthorizationTransaction, error) {
	if r.transaction == nil {
		return nil, transactiondomain.ErrNotFound
	}
	return r.transaction, nil
}

func (r *registrationTransactionRepository) Update(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
) error {
	r.updated = transaction
	return nil
}

type registrationIdentityRepository struct {
	created   *identitydomain.Identity
	createErr error
}

func (r *registrationIdentityRepository) Create(_ context.Context, identity *identitydomain.Identity) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = identity
	return nil
}

func (*registrationIdentityRepository) Get(context.Context, identitydomain.ID) (*identitydomain.Identity, error) {
	return nil, identitydomain.ErrNotFound
}

func (*registrationIdentityRepository) GetByEmail(context.Context, identitydomain.EmailAddress) (*identitydomain.Identity, error) {
	return nil, identitydomain.ErrNotFound
}

func (*registrationIdentityRepository) GetByPhone(context.Context, identitydomain.PhoneNumber) (*identitydomain.Identity, error) {
	return nil, identitydomain.ErrNotFound
}

func (*registrationIdentityRepository) Update(context.Context, *identitydomain.Identity) error {
	return nil
}

type registrationProfileRepository struct {
	created *identityprofiledomain.Profile
}

func (r *registrationProfileRepository) Create(_ context.Context, profile *identityprofiledomain.Profile) error {
	r.created = profile
	return nil
}

func (*registrationProfileRepository) Get(context.Context, identitydomain.ID) (*identityprofiledomain.Profile, error) {
	return nil, identityprofiledomain.ErrNotFound
}

type registrationPasswordRepository struct {
	created *passworddomain.Password
}

func (r *registrationPasswordRepository) Create(_ context.Context, password *passworddomain.Password) error {
	r.created = password
	return nil
}

func (*registrationPasswordRepository) Get(context.Context, identitydomain.ID) (*passworddomain.Password, error) {
	return nil, passworddomain.ErrNotFound
}

func (*registrationPasswordRepository) Update(context.Context, *passworddomain.Password) error {
	return nil
}

type registrationVerificationIssuer struct {
	challengeID  uuid.UUID
	expiresAt    time.Time
	emailCalls   int
	phoneCalls   int
	emailID      identitydomain.EmailID
	emailAddress identitydomain.EmailAddress
	phoneID      identitydomain.PhoneID
	phoneNumber  identitydomain.PhoneNumber
}

func (s *registrationVerificationIssuer) StartEmail(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	email identitydomain.Email,
	_ *sessiondomain.IPAddress,
	now time.Time,
) error {
	s.emailCalls++
	s.emailID = email.ID
	s.emailAddress = email.Address
	return transaction.BindEmailVerification(email.ID, s.challengeID, s.expiresAt, now)
}

func (s *registrationVerificationIssuer) StartPhone(
	_ context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	phone identitydomain.Phone,
	_ *sessiondomain.IPAddress,
	now time.Time,
) error {
	s.phoneCalls++
	s.phoneID = phone.ID
	s.phoneNumber = phone.Number
	return transaction.BindPhoneVerification(phone.ID, s.challengeID, s.expiresAt, now)
}

func (*registrationVerificationIssuer) VerifyEmail(
	context.Context,
	*transactiondomain.AuthorizationTransaction,
	string,
	*sessiondomain.IPAddress,
	time.Time,
) (identitydomain.EmailID, error) {
	return identitydomain.EmailID{}, nil
}

func (*registrationVerificationIssuer) VerifyPhone(
	context.Context,
	*transactiondomain.AuthorizationTransaction,
	string,
	*sessiondomain.IPAddress,
	time.Time,
) (identitydomain.PhoneID, error) {
	return identitydomain.PhoneID{}, nil
}

type registrationPasswordHasher struct{}

func (registrationPasswordHasher) Hash(string) (string, error) {
	return "encoded-password", nil
}

type registrationTransactionManager struct {
	calls int
}

func (m *registrationTransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type registrationIDGenerator struct {
	values []uuid.UUID
	index  int
}

func (g *registrationIDGenerator) New() uuid.UUID {
	value := g.values[g.index]
	g.index++
	return value
}

type registrationClock struct {
	now time.Time
}

func (c registrationClock) Now() time.Time { return c.now }
