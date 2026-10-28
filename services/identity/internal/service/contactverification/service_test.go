package contactverification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	emailchallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/emailverificationchallenge"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	phonechallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/phoneverificationchallenge"
	"github.com/stretchr/testify/require"
)

func TestServiceIssuesEmailChallenge(t *testing.T) {
	t.Parallel()

	now := contactVerificationTestTime()
	challengeID := uuid.New()
	email := identitydomain.Email{ID: identitydomain.EmailID(uuid.New()), Address: "user@example.com"}
	emails := &emailRepositoryStub{}
	service := newContactVerificationServiceForTest(t, emails, &phoneRepositoryStub{}, challengeID)
	transaction := contactVerificationTransaction(now, transactiondomain.NextStepEmailVerification)

	err := service.StartEmail(t.Context(), transaction, email, nil, now)

	require.NoError(t, err)
	require.Equal(t, &transactiondomain.VerificationReference{
		Channel:     transactiondomain.VerificationChannelEmail,
		ContactID:   email.ID.UUID(),
		ChallengeID: challengeID,
		ExpiresAt:   now.Add(10 * time.Minute),
	}, transaction.Verification)
	require.Equal(t, email.ID, emails.revokedEmailID)
	require.Equal(t, now, emails.revokedAt)
	require.NotNil(t, emails.created)
	require.Equal(t, emailchallengedomain.ID(challengeID), emails.created.ID)
	require.Equal(t, email.ID, emails.created.EmailID)
	require.Equal(t, emailchallengedomain.TokenHash("digest:123456"), emails.created.TokenHash)
	require.Equal(t, 3, emails.created.MaxAttempts)
	messages := service.outbox.(*outboxWriterStub).messages
	require.Len(t, messages, 1)
	require.Equal(t, emailchallengedomain.EventTypeIssued, messages[0].EventType)
	var payload emailchallengedomain.IssuedEvent
	require.NoError(t, json.Unmarshal(messages[0].Payload, &payload))
	require.Equal(t, email.Address, payload.Email)
	require.Equal(t, "123456", payload.Code)
}

func TestServiceIssuesPhoneChallenge(t *testing.T) {
	t.Parallel()

	now := contactVerificationTestTime()
	challengeID := uuid.New()
	phone := identitydomain.Phone{ID: identitydomain.PhoneID(uuid.New()), Number: "+79991234567"}
	phones := &phoneRepositoryStub{}
	service := newContactVerificationServiceForTest(t, &emailRepositoryStub{}, phones, challengeID)
	transaction := contactVerificationTransaction(now, transactiondomain.NextStepPhoneVerification)

	err := service.StartPhone(t.Context(), transaction, phone, nil, now)

	require.NoError(t, err)
	require.Equal(t, challengeID, transaction.Verification.ChallengeID)
	require.Equal(t, phone.ID, phones.revokedPhoneID)
	require.NotNil(t, phones.created)
	require.Equal(t, phonechallengedomain.TokenHash("digest:123456"), phones.created.TokenHash)
	messages := service.outbox.(*outboxWriterStub).messages
	require.Len(t, messages, 1)
	require.Equal(t, phonechallengedomain.EventTypeIssued, messages[0].EventType)
	var payload phonechallengedomain.IssuedEvent
	require.NoError(t, json.Unmarshal(messages[0].Payload, &payload))
	require.Equal(t, phone.Number, payload.Phone)
	require.Equal(t, "123456", payload.Code)
}

func TestServiceRejectsMalformedGeneratedCode(t *testing.T) {
	t.Parallel()

	service, err := NewService(
		Config{Lifetime: 10 * time.Minute, MaxAttempts: 3},
		&emailRepositoryStub{},
		&phoneRepositoryStub{},
		&outboxWriterStub{},
		malformedCodeGeneratorStub{},
		tokenDigesterStub{},
		idGeneratorStub{value: uuid.New()},
	)
	require.NoError(t, err)

	err = service.StartEmail(
		t.Context(),
		contactVerificationTransaction(contactVerificationTestTime(), transactiondomain.NextStepEmailVerification),
		identitydomain.Email{ID: identitydomain.EmailID(uuid.New()), Address: "user@example.com"},
		nil,
		contactVerificationTestTime(),
	)

	require.ErrorContains(t, err, "exactly six digits")
}

func TestServiceVerifiesEmailChallenge(t *testing.T) {
	t.Parallel()

	now := contactVerificationTestTime()
	challengeID := emailchallengedomain.ID(uuid.New())
	emailID := identitydomain.EmailID(uuid.New())
	challenge, err := emailchallengedomain.New(
		challengeID,
		emailID,
		emailchallengedomain.TokenHash("digest:123456"),
		3,
		now.Add(10*time.Minute),
		nil,
		now,
	)
	require.NoError(t, err)
	emails := &emailRepositoryStub{loaded: challenge}
	service := newContactVerificationServiceForTest(t, emails, &phoneRepositoryStub{}, uuid.New())
	transaction := contactVerificationTransaction(now, transactiondomain.NextStepEmailVerification)
	require.NoError(t, transaction.BindEmailVerification(emailID, challengeID.UUID(), challenge.ExpiresAt, now))

	verifiedEmailID, err := service.VerifyEmail(t.Context(), transaction, "000000", nil, now.Add(time.Minute))

	require.NoError(t, err)
	require.Equal(t, emailID, verifiedEmailID)
	require.Same(t, challenge, emails.updated)
	require.NotNil(t, challenge.ConsumedAt)
	require.Equal(t, 1, challenge.Attempts)
}

func TestServiceVerifiesPhoneChallenge(t *testing.T) {
	t.Parallel()

	now := contactVerificationTestTime()
	challengeID := phonechallengedomain.ID(uuid.New())
	phoneID := identitydomain.PhoneID(uuid.New())
	challenge, err := phonechallengedomain.New(
		challengeID,
		phoneID,
		phonechallengedomain.TokenHash("digest:123456"),
		3,
		now.Add(10*time.Minute),
		nil,
		now,
	)
	require.NoError(t, err)
	phones := &phoneRepositoryStub{loaded: challenge}
	service := newContactVerificationServiceForTest(t, &emailRepositoryStub{}, phones, uuid.New())
	transaction := contactVerificationTransaction(now, transactiondomain.NextStepPhoneVerification)
	require.NoError(t, transaction.BindPhoneVerification(phoneID, challengeID.UUID(), challenge.ExpiresAt, now))

	verifiedPhoneID, err := service.VerifyPhone(t.Context(), transaction, "999999", nil, now.Add(time.Minute))

	require.NoError(t, err)
	require.Equal(t, phoneID, verifiedPhoneID)
	require.Same(t, challenge, phones.updated)
	require.NotNil(t, challenge.ConsumedAt)
}

func TestServiceRejectsEmailVerificationForAnotherTransactionStep(t *testing.T) {
	t.Parallel()

	now := contactVerificationTestTime()
	emails := &emailRepositoryStub{}
	service := newContactVerificationServiceForTest(t, emails, &phoneRepositoryStub{}, uuid.New())
	transaction := contactVerificationTransaction(now, transactiondomain.NextStepConsent)

	_, err := service.VerifyEmail(t.Context(), transaction, "000000", nil, now)

	require.ErrorIs(t, err, ErrChallengeUnavailable)
	require.Nil(t, emails.updated)
}

func TestNewServiceValidatesDependencies(t *testing.T) {
	t.Parallel()

	cfg := Config{Lifetime: 10 * time.Minute, MaxAttempts: 3}
	emails := &emailRepositoryStub{}
	phones := &phoneRepositoryStub{}
	tokens := tokenGeneratorStub{}
	digester := tokenDigesterStub{}
	ids := idGeneratorStub{value: uuid.New()}
	outbox := &outboxWriterStub{}

	_, err := NewService(Config{}, emails, phones, outbox, tokens, digester, ids)
	require.ErrorContains(t, err, "validate contact verification config")
	_, err = NewService(cfg, nil, phones, outbox, tokens, digester, ids)
	require.ErrorContains(t, err, "email challenge repository is required")
	_, err = NewService(cfg, emails, nil, outbox, tokens, digester, ids)
	require.ErrorContains(t, err, "phone challenge repository is required")
	_, err = NewService(cfg, emails, phones, nil, tokens, digester, ids)
	require.ErrorContains(t, err, "outbox writer is required")
	_, err = NewService(cfg, emails, phones, outbox, nil, digester, ids)
	require.ErrorContains(t, err, "code generator is required")
	_, err = NewService(cfg, emails, phones, outbox, tokens, nil, ids)
	require.ErrorContains(t, err, "token digester is required")
	_, err = NewService(cfg, emails, phones, outbox, tokens, digester, nil)
	require.ErrorContains(t, err, "ID generator is required")
}

type emailRepositoryStub struct {
	revokedEmailID identitydomain.EmailID
	revokedAt      time.Time
	created        *emailchallengedomain.EmailVerificationChallenge
	loaded         *emailchallengedomain.EmailVerificationChallenge
	updated        *emailchallengedomain.EmailVerificationChallenge
}

func (s *emailRepositoryStub) RevokeActiveByEmailID(
	_ context.Context,
	emailID identitydomain.EmailID,
	revokedAt time.Time,
) (int64, error) {
	s.revokedEmailID = emailID
	s.revokedAt = revokedAt
	return 1, nil
}

func (s *emailRepositoryStub) Create(_ context.Context, challenge *emailchallengedomain.EmailVerificationChallenge) error {
	s.created = challenge
	return nil
}

func (s *emailRepositoryStub) GetForUpdate(
	_ context.Context,
	_ emailchallengedomain.ID,
) (*emailchallengedomain.EmailVerificationChallenge, error) {
	return s.loaded, nil
}

func (s *emailRepositoryStub) Update(_ context.Context, challenge *emailchallengedomain.EmailVerificationChallenge) error {
	s.updated = challenge
	return nil
}

type phoneRepositoryStub struct {
	revokedPhoneID identitydomain.PhoneID
	created        *phonechallengedomain.PhoneVerificationChallenge
	loaded         *phonechallengedomain.PhoneVerificationChallenge
	updated        *phonechallengedomain.PhoneVerificationChallenge
}

func (s *phoneRepositoryStub) RevokeActiveByPhoneID(
	_ context.Context,
	phoneID identitydomain.PhoneID,
	_ time.Time,
) (int64, error) {
	s.revokedPhoneID = phoneID
	return 1, nil
}

func (s *phoneRepositoryStub) Create(_ context.Context, challenge *phonechallengedomain.PhoneVerificationChallenge) error {
	s.created = challenge
	return nil
}

func (s *phoneRepositoryStub) GetForUpdate(
	_ context.Context,
	_ phonechallengedomain.ID,
) (*phonechallengedomain.PhoneVerificationChallenge, error) {
	return s.loaded, nil
}

func (s *phoneRepositoryStub) Update(_ context.Context, challenge *phonechallengedomain.PhoneVerificationChallenge) error {
	s.updated = challenge
	return nil
}

type tokenGeneratorStub struct{}

func (tokenGeneratorStub) Generate() (string, error) { return "123456", nil }

type malformedCodeGeneratorStub struct{}

func (malformedCodeGeneratorStub) Generate() (string, error) { return "invalid", nil }

type tokenDigesterStub struct{}

func (tokenDigesterStub) Digest(value string) string { return "digest:" + value }

type idGeneratorStub struct{ value uuid.UUID }

func (s idGeneratorStub) New() uuid.UUID { return s.value }

type outboxWriterStub struct {
	messages []eventdomain.Message
}

func (s *outboxWriterStub) Create(_ context.Context, message eventdomain.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func newContactVerificationServiceForTest(
	t *testing.T,
	emails EmailChallengeRepository,
	phones PhoneChallengeRepository,
	challengeID uuid.UUID,
) *Service {
	t.Helper()
	service, err := NewService(
		Config{Lifetime: 10 * time.Minute, MaxAttempts: 3},
		emails,
		phones,
		&outboxWriterStub{},
		tokenGeneratorStub{},
		tokenDigesterStub{},
		idGeneratorStub{value: challengeID},
	)
	require.NoError(t, err)
	return service
}

func contactVerificationTestTime() time.Time {
	return time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
}

func contactVerificationTransaction(now time.Time, step transactiondomain.NextStep) *transactiondomain.AuthorizationTransaction {
	identityID := identitydomain.ID(uuid.New())
	return &transactiondomain.AuthorizationTransaction{
		Status:     transactiondomain.StatusPending,
		NextStep:   step,
		IdentityID: &identityID,
		ExpiresAt:  now.Add(15 * time.Minute),
	}
}
