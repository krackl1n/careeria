package oauthclientadmin

import (
	"context"
	"time"

	"github.com/google/uuid"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type Repository interface {
	Create(context.Context, *oauthclientdomain.OAuthClient) error
	GetByClientID(context.Context, oauthclientdomain.ClientID) (*oauthclientdomain.OAuthClient, error)
}

type TransactionManager interface {
	Do(context.Context, func(context.Context) error) error
}

type IDGenerator interface {
	New() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	clients Repository
	tx      TransactionManager
	ids     IDGenerator
	clock   Clock
}

func NewService(
	clients Repository,
	tx TransactionManager,
	ids IDGenerator,
	clock Clock,
) *Service {
	return &Service{
		clients: clients,
		tx:      tx,
		ids:     ids,
		clock:   clock,
	}
}
