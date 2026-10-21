package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	policydomain "github.com/krackl1n/careeria/services/hiring/internal/domain/policy"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *applicationdomain.JobApplication) error
	Update(ctx context.Context, app *applicationdomain.JobApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*applicationdomain.JobApplication, error)
	GetActiveByCandidateAndVacancy(ctx context.Context, candidateID, vacancyID uuid.UUID) (*applicationdomain.JobApplication, error)
}

type InterviewSummaryRepository interface {
	FindByApplicationID(ctx context.Context, appID uuid.UUID) ([]interviewdomain.InterviewSummary, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, message eventdomain.Message) error
}

type AuthorizationPort interface {
	CanApplyVacancy(ctx context.Context, actorID, vacancyID uuid.UUID) error
	CanManageApplication(ctx context.Context, actorID, appID uuid.UUID) error
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
	appRepo      ApplicationRepository
	interviewSum InterviewSummaryRepository
	outboxRepo   OutboxRepository
	authPort     AuthorizationPort
	offerPolicy  *policydomain.OfferEligibilityPolicy
	txManager    TransactionManager
	eventFactory eventdomain.Factory
	clock        Clock
	idGen        IDGenerator
}

func NewService(
	appRepo ApplicationRepository,
	interviewSum InterviewSummaryRepository,
	outboxRepo OutboxRepository,
	authPort AuthorizationPort,
	offerPolicy *policydomain.OfferEligibilityPolicy,
	txManager TransactionManager,
	eventFactory eventdomain.Factory,
	clock Clock,
	idGen IDGenerator,
) *Service {
	return &Service{
		appRepo:      appRepo,
		interviewSum: interviewSum,
		outboxRepo:   outboxRepo,
		authPort:     authPort,
		offerPolicy:  offerPolicy,
		txManager:    txManager,
		eventFactory: eventFactory,
		clock:        clock,
		idGen:        idGen,
	}
}

