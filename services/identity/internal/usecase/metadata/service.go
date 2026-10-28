package metadata

import (
	"context"
	"errors"
	"fmt"
	"time"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type SigningKeyRepository interface {
	ListPublic(context.Context, time.Time) ([]*signingkeydomain.SigningKey, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	cfg   Config
	keys  SigningKeyRepository
	clock Clock
}

func NewService(cfg Config, keys SigningKeyRepository, clock Clock) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate metadata config: %w", err)
	}
	if keys == nil {
		return nil, errors.New("create metadata service: signing key repository is required")
	}
	if clock == nil {
		return nil, errors.New("create metadata service: clock is required")
	}

	return &Service{cfg: cfg, keys: keys, clock: clock}, nil
}
