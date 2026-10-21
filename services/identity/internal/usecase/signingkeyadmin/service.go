package signingkeyadmin

import (
	"context"
	"time"

	"github.com/google/uuid"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type Repository interface {
	Create(context.Context, *signingkeydomain.SigningKey) error
	GetByKeyID(context.Context, signingkeydomain.KeyID) (*signingkeydomain.SigningKey, error)
	Update(context.Context, *signingkeydomain.SigningKey) error
}

type IDGenerator interface {
	New() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

type KeyMaterial struct {
	KeyID         signingkeydomain.KeyID
	Algorithm     signingkeydomain.Algorithm
	PublicJWK     signingkeydomain.PublicJWK
	PrivateKeyRef signingkeydomain.PrivateKeyRef
}

type Service struct {
	keys     Repository
	ids      IDGenerator
	clock    Clock
	material KeyMaterial
}

func NewService(keys Repository, ids IDGenerator, clock Clock, material KeyMaterial) *Service {
	return &Service{keys: keys, ids: ids, clock: clock, material: material}
}
