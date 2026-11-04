package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmailLifecycleKeepsIdentityLoginInvariant(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	firstAddress := mustEmail(t, " First@Example.COM ")
	firstID := EmailID(uuid.New())
	entity, err := NewWithEmail(ID(uuid.New()), firstID, firstAddress, now)
	require.NoError(t, err)
	require.Equal(t, StatusPending, entity.Status)
	require.Equal(t, EmailAddress("first@example.com"), entity.Emails[0].Address)
	require.True(t, entity.Emails[0].IsPrimary)

	err = entity.Activate(now.Add(time.Minute))
	require.ErrorIs(t, err, ErrLoginMethodRequired)

	require.NoError(t, entity.VerifyEmail(firstID, now.Add(2*time.Minute)))
	require.NoError(t, entity.Activate(now.Add(3*time.Minute)))

	secondID := EmailID(uuid.New())
	require.NoError(t, entity.AddEmail(secondID, mustEmail(t, "second@example.com"), now.Add(4*time.Minute)))
	require.EqualError(t, entity.SetPrimaryEmail(secondID, now.Add(5*time.Minute)), "set primary identity email: email is not verified")
	require.EqualError(t, entity.RemoveEmail(firstID, now.Add(5*time.Minute)), "remove identity email: select another primary email first")

	require.NoError(t, entity.VerifyEmail(secondID, now.Add(5*time.Minute)))
	require.NoError(t, entity.SetPrimaryEmail(secondID, now.Add(6*time.Minute)))
	require.NoError(t, entity.RemoveEmail(firstID, now.Add(7*time.Minute)))

	emails := entity.Emails
	require.Len(t, emails, 2)
	require.NotNil(t, emails[0].DeletedAt)
	require.False(t, emails[0].IsPrimary)
	require.True(t, emails[1].IsPrimary)
	require.Equal(t, EmailAddress("first@example.com"), emails[0].Address)

	err = entity.RemoveEmail(secondID, now.Add(8*time.Minute))
	require.ErrorIs(t, err, ErrLoginMethodRequired)
	require.Nil(t, entity.Emails[1].DeletedAt, "failed mutation must not change aggregate state")
}

func TestExternalAccountCanKeepIdentityUsableWhenEmailIsRemoved(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	emailID := EmailID(uuid.New())
	entity, err := NewWithEmail(ID(uuid.New()), emailID, mustEmail(t, "user@example.com"), now)
	require.NoError(t, err)
	require.NoError(t, entity.VerifyEmail(emailID, now.Add(time.Minute)))
	require.NoError(t, entity.Activate(now.Add(2*time.Minute)))

	accountID := ExternalAccountID(uuid.New())
	require.NoError(t, entity.LinkExternalAccount(
		accountID,
		mustProvider(t, "google"),
		mustSubject(t, "provider-user-id"),
		now.Add(3*time.Minute),
	))
	require.NoError(t, entity.RemoveEmail(emailID, now.Add(4*time.Minute)))

	err = entity.DisableExternalAccount(accountID, now.Add(5*time.Minute))
	require.ErrorIs(t, err, ErrLoginMethodRequired)
	require.Nil(t, entity.ExternalAccounts[0].DisabledAt)
}

func TestExternalAccountLifecycle(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	firstID := ExternalAccountID(uuid.New())
	entity, err := NewWithExternalAccount(
		ID(uuid.New()),
		firstID,
		mustProvider(t, "github"),
		mustSubject(t, "42"),
		now,
	)
	require.NoError(t, err)
	require.NoError(t, entity.Activate(now.Add(time.Minute)))
	require.NoError(t, entity.RecordExternalAuthentication(firstID, now.Add(2*time.Minute)))

	secondID := ExternalAccountID(uuid.New())
	require.NoError(t, entity.LinkExternalAccount(
		secondID,
		mustProvider(t, "google"),
		mustSubject(t, "subject"),
		now.Add(3*time.Minute),
	))
	require.NoError(t, entity.DisableExternalAccount(firstID, now.Add(4*time.Minute)))
	require.ErrorIs(t, entity.RecordExternalAuthentication(firstID, now.Add(5*time.Minute)), ErrExternalAccountDisabled)
	require.NoError(t, entity.EnableExternalAccount(firstID, now.Add(5*time.Minute)))
	require.NoError(t, entity.UnlinkExternalAccount(secondID, now.Add(6*time.Minute)))
	require.ErrorIs(t, entity.UnlinkExternalAccount(firstID, now.Add(7*time.Minute)), ErrLoginMethodRequired)
}

func TestExternalAuthenticationRequiresActiveIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	accountID := ExternalAccountID(uuid.New())
	entity, err := NewWithExternalAccount(
		ID(uuid.New()),
		accountID,
		mustProvider(t, "google"),
		mustSubject(t, "subject"),
		now,
	)
	require.NoError(t, err)

	err = entity.RecordExternalAuthentication(accountID, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestPhoneLifecycleAndPrimarySelection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	firstID := PhoneID(uuid.New())
	entity, err := NewWithPhone(ID(uuid.New()), firstID, mustPhone(t, "+79991234567"), now)
	require.NoError(t, err)
	require.NoError(t, entity.VerifyPhone(firstID, now.Add(time.Minute)))
	require.NoError(t, entity.Activate(now.Add(2*time.Minute)))

	secondID := PhoneID(uuid.New())
	require.NoError(t, entity.AddPhone(secondID, mustPhone(t, "+12025550123"), now.Add(3*time.Minute)))
	require.EqualError(t, entity.RemovePhone(firstID, now.Add(4*time.Minute)), "remove identity phone: select another primary phone first")
	require.NoError(t, entity.VerifyPhone(secondID, now.Add(4*time.Minute)))
	require.NoError(t, entity.SetPrimaryPhone(secondID, now.Add(5*time.Minute)))
	require.NoError(t, entity.RemovePhone(firstID, now.Add(6*time.Minute)))
}

func TestDeletedIdentityRejectsFurtherChanges(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity, err := NewWithExternalAccount(
		ID(uuid.New()),
		ExternalAccountID(uuid.New()),
		mustProvider(t, "google"),
		mustSubject(t, "subject"),
		now,
	)
	require.NoError(t, err)
	require.NoError(t, entity.Delete(now.Add(time.Minute)))
	require.Equal(t, StatusDeleted, entity.Status)

	err = entity.AddEmail(EmailID(uuid.New()), mustEmail(t, "user@example.com"), now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
	require.NotNil(t, entity.DeletedAt)
}

func TestValueObjects(t *testing.T) {
	t.Parallel()

	email, err := NewEmailAddress(" User@Example.COM ")
	require.NoError(t, err)
	require.Equal(t, EmailAddress("user@example.com"), email)

	_, err = NewEmailAddress("not-an-email")
	require.Error(t, err)
	_, err = NewPhoneNumber("89991234567")
	require.Error(t, err)
	_, err = NewPhoneNumber("+012345678")
	require.Error(t, err)
	phone, err := NewPhoneNumber("+79991234567")
	require.NoError(t, err)
	require.Equal(t, PhoneNumber("+79991234567"), phone)
}

func TestMutationsRejectTimeGoingBackwards(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity, err := NewWithExternalAccount(
		ID(uuid.New()),
		ExternalAccountID(uuid.New()),
		mustProvider(t, "google"),
		mustSubject(t, "subject"),
		now,
	)
	require.NoError(t, err)

	err = entity.Disable(now.Add(-time.Second))
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrInvalidTransition))
}

func mustEmail(t *testing.T, value string) EmailAddress {
	t.Helper()
	address, err := NewEmailAddress(value)
	require.NoError(t, err)
	return address
}

func mustPhone(t *testing.T, value string) PhoneNumber {
	t.Helper()
	number, err := NewPhoneNumber(value)
	require.NoError(t, err)
	return number
}

func mustProvider(t *testing.T, value string) ExternalProvider {
	t.Helper()
	provider, err := NewExternalProvider(value)
	require.NoError(t, err)
	return provider
}

func mustSubject(t *testing.T, value string) ExternalSubject {
	t.Helper()
	subject, err := NewExternalSubject(value)
	require.NoError(t, err)
	return subject
}
