package signingkey

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSigningKeyLifecycle(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	key := newSigningKeyForTest(t, now.Add(time.Minute), nil, now)

	err := key.Activate(now.Add(30 * time.Second))
	require.ErrorIs(t, err, ErrNotYetValid)
	require.Equal(t, StatusPending, key.Status)

	activatedAt := now.Add(time.Minute)
	require.NoError(t, key.Activate(activatedAt))
	require.Equal(t, StatusActive, key.Status)
	require.Equal(t, activatedAt, *key.ActivatedAt)
	require.NoError(t, key.ValidateForSigning(activatedAt))
	require.NoError(t, key.ValidateForVerification(activatedAt))

	retiringAt := now.Add(2 * time.Minute)
	require.NoError(t, key.StartRetirement(retiringAt))
	require.Equal(t, StatusRetiring, key.Status)
	require.Equal(t, retiringAt, *key.RetiringAt)
	require.ErrorIs(t, key.ValidateForSigning(retiringAt), ErrInvalidStatus)
	require.NoError(t, key.ValidateForVerification(retiringAt))

	retiredAt := now.Add(3 * time.Minute)
	require.NoError(t, key.Retire(retiredAt))
	require.Equal(t, StatusRetired, key.Status)
	require.Equal(t, retiredAt, *key.RetiredAt)
	require.ErrorIs(t, key.ValidateForVerification(retiredAt), ErrInvalidStatus)
}

func TestActiveSigningKeyCanBeMarkedCompromised(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	key := newSigningKeyForTest(t, now, nil, now)
	require.NoError(t, key.Activate(now))

	compromisedAt := now.Add(time.Minute)
	require.NoError(t, key.MarkCompromised(compromisedAt))
	require.Equal(t, StatusCompromised, key.Status)
	require.Equal(t, compromisedAt, *key.CompromisedAt)
	require.ErrorIs(t, key.ValidateForSigning(compromisedAt), ErrCompromised)
	require.ErrorIs(t, key.ValidateForVerification(compromisedAt), ErrCompromised)

	require.NoError(t, key.MarkCompromised(now.Add(2*time.Minute)))
	require.Equal(t, compromisedAt, *key.CompromisedAt)
}

func TestExpiredSigningKeyCannotBeActivatedOrUsed(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	expiresAt := now.Add(time.Minute)
	key := newSigningKeyForTest(t, now, &expiresAt, now)

	err := key.Activate(expiresAt)
	require.ErrorIs(t, err, ErrExpired)
	require.Equal(t, StatusPending, key.Status)

	require.NoError(t, key.Activate(now.Add(30*time.Second)))
	err = key.ValidateForSigning(expiresAt)
	require.ErrorIs(t, err, ErrExpired)
	err = key.ValidateForVerification(expiresAt)
	require.ErrorIs(t, err, ErrExpired)
}

func TestSigningKeyRejectsPrivateJWKMaterial(t *testing.T) {
	t.Parallel()

	_, err := NewPublicJWK([]byte(`{"kty":"RSA","n":"public","e":"AQAB","d":"private"}`))
	require.Error(t, err)

	_, err = NewPublicJWK([]byte(`{"kty":"oct","k":"secret"}`))
	require.Error(t, err)
}

func TestSigningKeyRejectsMismatchedJWKMetadata(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	publicJWK, err := NewPublicJWK([]byte(`{"kty":"OKP","crv":"Ed25519","x":"public","kid":"another-key","alg":"EdDSA","use":"sig"}`))
	require.NoError(t, err)

	_, err = New(
		ID(uuid.New()),
		KeyID("key-1"),
		Algorithm("EdDSA"),
		publicJWK,
		PrivateKeyRef("kms://identity/key-1"),
		now,
		nil,
		now,
	)
	require.Error(t, err)
}

func TestSigningKeyRequiresVerifiablePublicJWKMetadata(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	tests := map[string]string{
		"missing key ID":            `{"kty":"OKP","crv":"Ed25519","x":"public","alg":"EdDSA","use":"sig"}`,
		"wrong key type":            `{"kty":"RSA","n":"public","e":"AQAB","kid":"key-1","alg":"EdDSA","use":"sig"}`,
		"missing public coordinate": `{"kty":"OKP","crv":"Ed25519","kid":"key-1","alg":"EdDSA","use":"sig"}`,
	}

	for name, raw := range tests {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			publicJWK, err := NewPublicJWK([]byte(raw))
			require.NoError(t, err)
			_, err = New(
				ID(uuid.New()),
				KeyID("key-1"),
				AlgorithmEdDSA,
				publicJWK,
				PrivateKeyRef("kms://identity/key-1"),
				now,
				nil,
				now,
			)
			require.Error(t, err)
		})
	}
}

func TestSigningKeyCopiesPublicJWK(t *testing.T) {
	t.Parallel()

	now := signingKeyTestTime()
	raw := publicJWKForTest()
	publicJWK, err := NewPublicJWK(raw)
	require.NoError(t, err)
	key, err := New(
		ID(uuid.New()),
		KeyID("key-1"),
		Algorithm("EdDSA"),
		publicJWK,
		PrivateKeyRef("kms://identity/key-1"),
		now,
		nil,
		now,
	)
	require.NoError(t, err)

	raw[0] = '['
	publicJWK[0] = '['
	require.Equal(t, byte('{'), key.PublicJWK[0])
}

func newSigningKeyForTest(t *testing.T, notBefore time.Time, expiresAt *time.Time, now time.Time) *SigningKey {
	t.Helper()

	publicJWK, err := NewPublicJWK(publicJWKForTest())
	require.NoError(t, err)
	key, err := New(
		ID(uuid.New()),
		KeyID("key-1"),
		Algorithm("EdDSA"),
		publicJWK,
		PrivateKeyRef("kms://identity/key-1"),
		notBefore,
		expiresAt,
		now,
	)
	require.NoError(t, err)
	return key
}

func publicJWKForTest() []byte {
	return []byte(`{"kty":"OKP","crv":"Ed25519","x":"public","kid":"key-1","alg":"EdDSA","use":"sig"}`)
}

func signingKeyTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
