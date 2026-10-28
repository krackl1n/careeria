package signingkeyadmin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

func TestCreateSigningKeyCreatesActiveConfiguredKey(t *testing.T) {
	repository := &keyRepository{}
	service := NewService(repository, keyUUIDGenerator{}, keyClock{}, configuredKeyMaterial())

	key, err := service.CreateSigningKey(context.Background())

	require.NoError(t, err)
	require.Equal(t, signingkeydomain.StatusActive, key.Status)
	require.Equal(t, configuredKeyMaterial().KeyID, key.KeyID)
	require.Equal(t, 1, repository.createCalls)
	require.Zero(t, repository.updateCalls)
}

func TestCreateSigningKeyIsIdempotentForSameActiveKey(t *testing.T) {
	repository := &keyRepository{}
	service := NewService(repository, keyUUIDGenerator{}, keyClock{}, configuredKeyMaterial())

	first, err := service.CreateSigningKey(context.Background())
	require.NoError(t, err)
	second, err := service.CreateSigningKey(context.Background())

	require.NoError(t, err)
	require.Same(t, first, second)
	require.Equal(t, 1, repository.createCalls)
	require.Zero(t, repository.updateCalls)
}

func TestCreateSigningKeyActivatesExistingPendingKey(t *testing.T) {
	material := configuredKeyMaterial()
	key, err := signingkeydomain.New(
		signingkeydomain.ID(uuid.New()),
		material.KeyID,
		material.Algorithm,
		material.PublicJWK,
		material.PrivateKeyRef,
		keyClock{}.Now(),
		nil,
		keyClock{}.Now(),
	)
	require.NoError(t, err)
	repository := &keyRepository{key: key}
	service := NewService(repository, keyUUIDGenerator{}, keyClock{}, material)

	result, err := service.CreateSigningKey(context.Background())

	require.NoError(t, err)
	require.Equal(t, signingkeydomain.StatusActive, result.Status)
	require.Zero(t, repository.createCalls)
	require.Equal(t, 1, repository.updateCalls)
}

func TestCreateSigningKeyRejectsDifferentStoredMaterial(t *testing.T) {
	material := configuredKeyMaterial()
	key, err := signingkeydomain.New(
		signingkeydomain.ID(uuid.New()),
		material.KeyID,
		material.Algorithm,
		material.PublicJWK,
		signingkeydomain.PrivateKeyRef("local://another-key"),
		keyClock{}.Now(),
		nil,
		keyClock{}.Now(),
	)
	require.NoError(t, err)
	repository := &keyRepository{key: key}
	service := NewService(repository, keyUUIDGenerator{}, keyClock{}, material)

	_, err = service.CreateSigningKey(context.Background())

	require.Error(t, err)
	require.True(t, apperr.IsType(err, apperr.TypeConflict))
	require.Zero(t, repository.createCalls)
	require.Zero(t, repository.updateCalls)
}

func configuredKeyMaterial() KeyMaterial {
	return KeyMaterial{
		KeyID:         signingkeydomain.KeyID("identity-local-1"),
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PublicJWK:     signingkeydomain.PublicJWK(`{"kty":"OKP","crv":"Ed25519","x":"public-key","kid":"identity-local-1","alg":"EdDSA","use":"sig"}`),
		PrivateKeyRef: signingkeydomain.PrivateKeyRef("local://access-token-signing-key"),
	}
}

type keyRepository struct {
	key         *signingkeydomain.SigningKey
	createCalls int
	updateCalls int
}

func (r *keyRepository) Create(_ context.Context, key *signingkeydomain.SigningKey) error {
	if r.key != nil {
		return signingkeydomain.ErrKeyIDAlreadyExists
	}
	r.key = key
	r.createCalls++
	return nil
}

func (r *keyRepository) GetByKeyID(_ context.Context, keyID signingkeydomain.KeyID) (*signingkeydomain.SigningKey, error) {
	if r.key == nil || r.key.KeyID != keyID {
		return nil, signingkeydomain.ErrNotFound
	}
	return r.key, nil
}

func (r *keyRepository) Update(_ context.Context, key *signingkeydomain.SigningKey) error {
	r.key = key
	r.updateCalls++
	return nil
}

type keyUUIDGenerator struct{}

func (keyUUIDGenerator) New() uuid.UUID { return uuid.New() }

type keyClock struct{}

func (keyClock) Now() time.Time {
	return time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
}
