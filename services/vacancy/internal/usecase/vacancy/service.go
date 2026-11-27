package vacancy

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/vacancy/internal/domain/event"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different input")
)

type Repository interface {
	Create(context.Context, *domain.Vacancy, uuid.UUID, string) error
	Get(context.Context, uuid.UUID) (*domain.Vacancy, error)
	GetByCreateRequest(context.Context, uuid.UUID) (*domain.Vacancy, string, error)
	Update(context.Context, *domain.Vacancy, int64) error
}
type Outbox interface {
	Create(context.Context, event.Message) error
}
type TransactionManager interface {
	Do(context.Context, func(context.Context) error) error
}
type Clock interface{ Now() time.Time }
type IDGenerator interface{ New() uuid.UUID }

type Service struct {
	repository   Repository
	outbox       Outbox
	transactions TransactionManager
	clock        Clock
	ids          IDGenerator
}

func NewService(repo Repository, outbox Outbox, transactions TransactionManager, clock Clock, ids IDGenerator) (*Service, error) {
	if repo == nil || outbox == nil || transactions == nil || clock == nil || ids == nil {
		return nil, errors.New("vacancy service dependencies are required")
	}
	return &Service{repository: repo, outbox: outbox, transactions: transactions, clock: clock, ids: ids}, nil
}
