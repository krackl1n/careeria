package idtoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type SigningKeyRepository interface {
	GetActive(context.Context, signingkeydomain.Algorithm, time.Time) (*signingkeydomain.SigningKey, error)
}

type SignInput struct {
	KeyID         signingkeydomain.KeyID
	Algorithm     signingkeydomain.Algorithm
	PrivateKeyRef signingkeydomain.PrivateKeyRef
	Claims        Claims
}

type Signer interface {
	SignIDToken(context.Context, SignInput) (string, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	cfg       Config
	algorithm signingkeydomain.Algorithm
	keys      SigningKeyRepository
	signer    Signer
	clock     Clock
}

func NewService(
	cfg Config,
	keys SigningKeyRepository,
	signer Signer,
	clock Clock,
) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate ID token config: %w", err)
	}
	if keys == nil {
		return nil, errors.New("create ID token service: signing key repository is required")
	}
	if signer == nil {
		return nil, errors.New("create ID token service: signer is required")
	}
	if clock == nil {
		return nil, errors.New("create ID token service: clock is required")
	}

	return &Service{
		cfg:       cfg,
		algorithm: signingkeydomain.Algorithm(cfg.Algorithm),
		keys:      keys,
		signer:    signer,
		clock:     clock,
	}, nil
}
