package mfa

import (
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestTOTPEnrollmentConfirmationAndReplayProtection(t *testing.T) {
	t.Parallel()

	now := mfaTestTime()
	entity := newMFAForTest(t)
	secret := EncryptedSecret("encrypted-secret")
	factorID := FactorID(uuid.New())

	require.NoError(t, entity.EnrollTOTP(
		factorID,
		stringPointer("  Authenticator  "),
		secret,
		EncryptionKeyRef("key://mfa/current"),
		TOTPAlgorithmSHA256,
		6,
		30,
		now,
	))
	secret[0] = 'X'
	require.Equal(t, "encrypted-secret", string(entity.Factors[0].TOTP.SecretEncrypted))
	require.Equal(t, "Authenticator", *entity.Factors[0].Name)
	require.False(t, entity.Factors[0].IsPrimary)

	err := entity.EnrollTOTP(
		FactorID(uuid.New()),
		nil,
		EncryptedSecret("another-secret"),
		EncryptionKeyRef("key://mfa/current"),
		TOTPAlgorithmSHA1,
		6,
		30,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrEnabledTOTPAlreadyExists)

	confirmedAt := now.Add(2 * time.Minute)
	require.NoError(t, entity.ConfirmFactor(factorID, confirmedAt))
	require.Equal(t, confirmedAt, *entity.Factors[0].ConfirmedAt)
	require.True(t, entity.Factors[0].IsPrimary)

	require.NoError(t, entity.RecordTOTPUse(factorID, 100, now.Add(3*time.Minute)))
	require.Equal(t, int64(100), *entity.Factors[0].TOTP.LastUsedTimeStep)
	require.ErrorIs(t, entity.RecordTOTPUse(factorID, 100, now.Add(4*time.Minute)), ErrTOTPReplay)
	require.Equal(t, int64(100), *entity.Factors[0].TOTP.LastUsedTimeStep)
}

func TestDisabledTOTPAllowsReplacementButCannotBeReenabledAlongsideIt(t *testing.T) {
	t.Parallel()

	now := mfaTestTime()
	entity := newMFAForTest(t)
	firstID := FactorID(uuid.New())
	require.NoError(t, enrollTestTOTP(entity, firstID, now))
	require.NoError(t, entity.ConfirmFactor(firstID, now.Add(time.Minute)))
	require.NoError(t, entity.DisableFactor(firstID, now.Add(2*time.Minute)))
	require.False(t, entity.Factors[0].IsPrimary)

	secondID := FactorID(uuid.New())
	require.NoError(t, enrollTestTOTP(entity, secondID, now.Add(3*time.Minute)))
	require.ErrorIs(t, entity.EnableFactor(firstID, now.Add(4*time.Minute)), ErrEnabledTOTPAlreadyExists)
	require.NotNil(t, entity.Factors[0].DisabledAt)
}

func TestWebAuthnEnrollmentAndPrimarySwitch(t *testing.T) {
	t.Parallel()

	now := mfaTestTime()
	entity := newMFAForTest(t)
	totpID := FactorID(uuid.New())
	require.NoError(t, enrollTestTOTP(entity, totpID, now))
	require.NoError(t, entity.ConfirmFactor(totpID, now.Add(time.Minute)))

	credentialID := CredentialID("credential-id")
	publicKey := PublicKey("public-key")
	webauthnID := FactorID(uuid.New())
	require.NoError(t, entity.EnrollWebAuthn(
		webauthnID,
		stringPointer("Security key"),
		credentialID,
		publicKey,
		1,
		nil,
		[]Transport{"usb", "nfc"},
		true,
		false,
		now.Add(2*time.Minute),
	))
	credentialID[0] = 'X'
	publicKey[0] = 'X'
	require.Equal(t, "credential-id", string(entity.Factors[1].WebAuthn.CredentialID))
	require.Equal(t, "public-key", string(entity.Factors[1].WebAuthn.PublicKey))

	require.NoError(t, entity.ConfirmFactor(webauthnID, now.Add(3*time.Minute)))
	require.NoError(t, entity.SetPrimaryFactor(webauthnID, now.Add(4*time.Minute)))
	require.False(t, entity.Factors[0].IsPrimary)
	require.True(t, entity.Factors[1].IsPrimary)
}

func TestWebAuthnSignatureCounterMustIncrease(t *testing.T) {
	t.Parallel()

	now := mfaTestTime()
	entity := newMFAForTest(t)
	factorID := FactorID(uuid.New())
	require.NoError(t, entity.EnrollWebAuthn(
		factorID,
		nil,
		CredentialID("credential-id"),
		PublicKey("public-key"),
		1,
		nil,
		nil,
		false,
		false,
		now,
	))
	require.NoError(t, entity.ConfirmFactor(factorID, now.Add(time.Minute)))
	require.NoError(t, entity.RecordWebAuthnUse(factorID, 2, false, now.Add(2*time.Minute)))
	require.Equal(t, int64(2), entity.Factors[0].WebAuthn.SignCount)

	err := entity.RecordWebAuthnUse(factorID, 2, false, now.Add(3*time.Minute))
	require.ErrorIs(t, err, ErrWebAuthnCounterRollback)
	require.Equal(t, int64(2), entity.Factors[0].WebAuthn.SignCount)
}

func TestUsingUnconfirmedFactorIsRejected(t *testing.T) {
	t.Parallel()

	now := mfaTestTime()
	entity := newMFAForTest(t)
	factorID := FactorID(uuid.New())
	require.NoError(t, enrollTestTOTP(entity, factorID, now))

	err := entity.RecordTOTPUse(factorID, 1, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrFactorNotConfirmed)
}

func newMFAForTest(t *testing.T) *MFA {
	t.Helper()

	entity, err := New(identitydomain.ID(uuid.New()))
	require.NoError(t, err)
	return entity
}

func enrollTestTOTP(entity *MFA, id FactorID, now time.Time) error {
	return entity.EnrollTOTP(
		id,
		nil,
		EncryptedSecret("encrypted-secret"),
		EncryptionKeyRef("key://mfa/current"),
		TOTPAlgorithmSHA1,
		6,
		30,
		now,
	)
}

func stringPointer(value string) *string {
	return &value
}

func mfaTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
