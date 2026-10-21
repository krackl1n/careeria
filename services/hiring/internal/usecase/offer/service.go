package offer

import (
	"context"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

type OfferRepository interface {
	Create(ctx context.Context, offer *offerdomain.JobOffer) error
	Update(ctx context.Context, offer *offerdomain.JobOffer) error
	GetByID(ctx context.Context, id uuid.UUID) (*offerdomain.JobOffer, error)
	GetByApplicationID(ctx context.Context, appID uuid.UUID) (*offerdomain.JobOffer, error)
}

type ApplicationCoordinator interface {
	GetApplication(ctx context.Context, appID uuid.UUID) (*applicationdomain.JobApplication, error)
	AdvanceToOfferAccepted(ctx context.Context, appID, actorID uuid.UUID, now time.Time) error
	RevertToInterview(ctx context.Context, appID, actorID uuid.UUID, reason string, now time.Time) error
}

type OutboxRepository interface {
	Save(ctx context.Context, message eventdomain.Message) error
}

type AuthorizationPort interface {
	CanManageOffer(ctx context.Context, actorID, appID uuid.UUID) error
	CanRespondToOffer(ctx context.Context, actorID, appID uuid.UUID) error
}

type TransactionManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	New() uuid.UUID
}

type Service struct {
	offerRepo    OfferRepository
	appCoord     ApplicationCoordinator
	outboxRepo   OutboxRepository
	authPort     AuthorizationPort
	txManager    TransactionManager
	eventFactory eventdomain.Factory
	clock        Clock
	idGen        IDGenerator
}

func NewService(
	offerRepo OfferRepository,
	appCoord ApplicationCoordinator,
	outboxRepo OutboxRepository,
	authPort AuthorizationPort,
	txManager TransactionManager,
	eventFactory eventdomain.Factory,
	clock Clock,
	idGen IDGenerator,
) *Service {
	return &Service{
		offerRepo:    offerRepo,
		appCoord:     appCoord,
		outboxRepo:   outboxRepo,
		authPort:     authPort,
		txManager:    txManager,
		eventFactory: eventFactory,
		clock:        clock,
		idGen:        idGen,
	}
}

