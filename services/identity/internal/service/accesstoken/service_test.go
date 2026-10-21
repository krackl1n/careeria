package accesstoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

const signedTokenForTest = "eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl"

func TestIssueBuildsJWTAccessTokenWithActiveSigningKey(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime().Add(789 * time.Millisecond)
	tokenID := uuid.New()
	key := activeSigningKeyForTest(now)
	keys := &signingKeyRepositoryStub{key: key}
	signer := &signerStub{value: signedTokenForTest}
	service := newServiceForTest(t, keys, signer, tokenID, now)
	input := validIssueInput(now)

	issued, err := service.Issue(t.Context(), input)
	require.NoError(t, err)

	issuedAt := time.Unix(now.Unix(), 0).UTC()
	require.Equal(t, IssuedToken{
		Value:     signedTokenForTest,
		ID:        tokenID,
		IssuedAt:  issuedAt,
		ExpiresAt: issuedAt.Add(15 * time.Minute),
	}, issued)
	require.Equal(t, 1, keys.calls)
	require.Equal(t, signingkeydomain.AlgorithmEdDSA, keys.algorithm)
	require.Equal(t, now, keys.now)
	require.Equal(t, 1, signer.calls)
	require.Equal(t, SignInput{
		KeyID:         signingkeydomain.KeyID("identity-2026-08"),
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PrivateKeyRef: signingkeydomain.PrivateKeyRef("vault://identity/signing/2026-08"),
		Claims: Claims{
			Issuer:               "https://identity.example.com",
			Subject:              input.IdentityID.UUID().String(),
			Audience:             []string{"https://api.example.com", "careeria"},
			ExpiresAt:            issued.ExpiresAt,
			IssuedAt:             issued.IssuedAt,
			TokenID:              tokenID.String(),
			ClientID:             "web-client",
			Scopes:               []string{"openid", "profile"},
			SessionID:            input.SessionID.UUID().String(),
			AuthenticationLevel:  "multi_factor",
			AuthenticatedMethods: []string{"password", "totp"},
			AuthenticatedAt:      input.AuthenticatedAt,
		},
	}, signer.input)
}

func TestIssuePropagatesSigningKeyRepositoryError(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime()
	repositoryErr := errors.New("database unavailable")
	keys := &signingKeyRepositoryStub{err: repositoryErr}
	signer := &signerStub{}
	service := newServiceForTest(t, keys, signer, uuid.New(), now)

	_, err := service.Issue(t.Context(), validIssueInput(now))
	require.ErrorIs(t, err, repositoryErr)
	require.Zero(t, signer.calls)
}

func TestIssueRejectsSigningKeyThatExpiresBeforeToken(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime()
	key := activeSigningKeyForTest(now)
	expiresAt := now.Add(5 * time.Minute)
	key.ExpiresAt = &expiresAt
	keys := &signingKeyRepositoryStub{key: key}
	signer := &signerStub{}
	service := newServiceForTest(t, keys, signer, uuid.New(), now)

	_, err := service.Issue(t.Context(), validIssueInput(now))
	require.ErrorContains(t, err, "signing key expires before access token")
	require.Zero(t, signer.calls)
}

func TestIssuePropagatesSignerError(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime()
	signerErr := errors.New("KMS unavailable")
	service := newServiceForTest(
		t,
		&signingKeyRepositoryStub{key: activeSigningKeyForTest(now)},
		&signerStub{err: signerErr},
		uuid.New(),
		now,
	)

	_, err := service.Issue(t.Context(), validIssueInput(now))
	require.ErrorIs(t, err, signerErr)
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	valid := Config{
		Issuer:    "https://identity.example.com",
		Audience:  []string{"https://api.example.com"},
		Algorithm: string(signingkeydomain.AlgorithmEdDSA),
		Lifetime:  15 * time.Minute,
	}
	tests := map[string]func(*Config){
		"issuer":    func(cfg *Config) { cfg.Issuer = "" },
		"audience":  func(cfg *Config) { cfg.Audience = nil },
		"algorithm": func(cfg *Config) { cfg.Algorithm = "HS256" },
		"lifetime":  func(cfg *Config) { cfg.Lifetime = 0 },
		"subsecond lifetime": func(cfg *Config) {
			cfg.Lifetime = time.Second + time.Millisecond
		},
	}

	require.NoError(t, valid.Validate())
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			mutate(&cfg)
			require.Error(t, cfg.Validate())
		})
	}
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Issuer:    "https://identity.example.com",
		Audience:  []string{"https://api.example.com"},
		Algorithm: string(signingkeydomain.AlgorithmEdDSA),
		Lifetime:  15 * time.Minute,
	}
	keys := &signingKeyRepositoryStub{}
	signer := &signerStub{}
	verifier := &verifierStub{}
	ids := &idGeneratorStub{value: uuid.New()}
	clock := &clockStub{value: accessTokenTestTime()}

	_, err := NewService(cfg, nil, signer, verifier, ids, clock)
	require.ErrorContains(t, err, "signing key repository is required")
	_, err = NewService(cfg, keys, nil, verifier, ids, clock)
	require.ErrorContains(t, err, "signer is required")
	_, err = NewService(cfg, keys, signer, nil, ids, clock)
	require.ErrorContains(t, err, "verifier is required")
	_, err = NewService(cfg, keys, signer, verifier, nil, clock)
	require.ErrorContains(t, err, "ID generator is required")
	_, err = NewService(cfg, keys, signer, verifier, ids, nil)
	require.ErrorContains(t, err, "clock is required")
}

func validIssueInput(now time.Time) IssueInput {
	return IssueInput{
		IdentityID:          uuidToIdentityID(uuid.New()),
		SessionID:           uuidToSessionID(uuid.New()),
		ClientID:            oauthclientdomain.ClientID("web-client"),
		GrantedScopes:       []oauthdomain.Scope{"openid", "profile"},
		AuthenticationLevel: authenticationdomain.LevelMultiFactor,
		AuthenticatedMethods: []authenticationdomain.Method{
			authenticationdomain.MethodPassword,
			authenticationdomain.MethodTOTP,
		},
		AuthenticatedAt: now.Add(-time.Minute),
	}
}

func activeSigningKeyForTest(now time.Time) *signingkeydomain.SigningKey {
	createdAt := now.Add(-time.Hour)
	activatedAt := now.Add(-30 * time.Minute)
	return &signingkeydomain.SigningKey{
		ID:            signingkeydomain.ID(uuid.New()),
		KeyID:         signingkeydomain.KeyID("identity-2026-08"),
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PublicJWK:     signingkeydomain.PublicJWK(`{"kty":"OKP","crv":"Ed25519","x":"public","kid":"identity-2026-08","alg":"EdDSA","use":"sig"}`),
		PrivateKeyRef: signingkeydomain.PrivateKeyRef("vault://identity/signing/2026-08"),
		Status:        signingkeydomain.StatusActive,
		CreatedAt:     createdAt,
		NotBefore:     createdAt,
		ActivatedAt:   &activatedAt,
	}
}

func newServiceForTest(
	t *testing.T,
	keys SigningKeyRepository,
	signer Signer,
	tokenID uuid.UUID,
	now time.Time,
) *Service {
	t.Helper()

	service, err := NewService(
		Config{
			Issuer:    "https://identity.example.com",
			Audience:  []string{"https://api.example.com", "careeria"},
			Algorithm: string(signingkeydomain.AlgorithmEdDSA),
			Lifetime:  15 * time.Minute,
		},
		keys,
		signer,
		&verifierStub{},
		&idGeneratorStub{value: tokenID},
		&clockStub{value: now},
	)
	require.NoError(t, err)
	return service
}

func uuidToIdentityID(value uuid.UUID) identitydomain.ID {
	return identitydomain.ID(value)
}

func uuidToSessionID(value uuid.UUID) sessiondomain.ID {
	return sessiondomain.ID(value)
}

type signingKeyRepositoryStub struct {
	key       *signingkeydomain.SigningKey
	err       error
	calls     int
	getCalls  int
	algorithm signingkeydomain.Algorithm
	keyID     signingkeydomain.KeyID
	now       time.Time
}

func (repository *signingKeyRepositoryStub) GetActive(
	_ context.Context,
	algorithm signingkeydomain.Algorithm,
	now time.Time,
) (*signingkeydomain.SigningKey, error) {
	repository.calls++
	repository.algorithm = algorithm
	repository.now = now
	return repository.key, repository.err
}

func (repository *signingKeyRepositoryStub) GetByKeyID(
	_ context.Context,
	keyID signingkeydomain.KeyID,
) (*signingkeydomain.SigningKey, error) {
	repository.getCalls++
	repository.keyID = keyID
	return repository.key, repository.err
}

type signerStub struct {
	value string
	err   error
	calls int
	input SignInput
}

func (signer *signerStub) Sign(_ context.Context, input SignInput) (string, error) {
	signer.calls++
	signer.input = input
	return signer.value, signer.err
}

type verifierStub struct {
	keyID       signingkeydomain.KeyID
	keyIDErr    error
	claims      Claims
	verifyErr   error
	keyIDCalls  int
	verifyCalls int
	value       string
	input       VerifyInput
}

func (verifier *verifierStub) KeyID(value string) (signingkeydomain.KeyID, error) {
	verifier.keyIDCalls++
	verifier.value = value
	return verifier.keyID, verifier.keyIDErr
}

func (verifier *verifierStub) Verify(input VerifyInput) (Claims, error) {
	verifier.verifyCalls++
	verifier.input = input
	return verifier.claims, verifier.verifyErr
}

type idGeneratorStub struct{ value uuid.UUID }

func (generator *idGeneratorStub) New() uuid.UUID { return generator.value }

type clockStub struct{ value time.Time }

func (clock *clockStub) Now() time.Time { return clock.value }

func accessTokenTestTime() time.Time {
	return time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
}
