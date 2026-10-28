package accesstoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type SigningKeyRepository interface {
	GetActive(context.Context, signingkeydomain.Algorithm, time.Time) (*signingkeydomain.SigningKey, error)
	GetByKeyID(context.Context, signingkeydomain.KeyID) (*signingkeydomain.SigningKey, error)
}

type SignInput struct {
	KeyID         signingkeydomain.KeyID
	Algorithm     signingkeydomain.Algorithm
	PrivateKeyRef signingkeydomain.PrivateKeyRef
	Claims        Claims
}

type Signer interface {
	Sign(context.Context, SignInput) (string, error)
}

type VerifyInput struct {
	Value     string
	KeyID     signingkeydomain.KeyID
	Algorithm signingkeydomain.Algorithm
	PublicJWK signingkeydomain.PublicJWK
	Issuer    string
	Audience  []string
	Now       time.Time
}

type Verifier interface {
	KeyID(string) (signingkeydomain.KeyID, error)
	Verify(VerifyInput) (Claims, error)
}

type IDGenerator interface {
	New() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	cfg       Config
	algorithm signingkeydomain.Algorithm
	keys      SigningKeyRepository
	signer    Signer
	verifier  Verifier
	ids       IDGenerator
	clock     Clock
}

func NewService(
	cfg Config,
	keys SigningKeyRepository,
	signer Signer,
	verifier Verifier,
	ids IDGenerator,
	clock Clock,
) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate access token config: %w", err)
	}
	if keys == nil {
		return nil, errors.New("create access token service: signing key repository is required")
	}
	if signer == nil {
		return nil, errors.New("create access token service: signer is required")
	}
	if verifier == nil {
		return nil, errors.New("create access token service: verifier is required")
	}
	if ids == nil {
		return nil, errors.New("create access token service: ID generator is required")
	}
	if clock == nil {
		return nil, errors.New("create access token service: clock is required")
	}

	return &Service{
		cfg:       cfg,
		algorithm: signingkeydomain.Algorithm(cfg.Algorithm),
		keys:      keys,
		signer:    signer,
		verifier:  verifier,
		ids:       ids,
		clock:     clock,
	}, nil
}
