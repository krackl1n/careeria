package interview

import (
	"context"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/krackl1n/careeria/services/hiring/internal/usecase/interview/room"
)

type InterviewRepository interface {
	Create(ctx context.Context, it *interviewdomain.Interview) error
	Update(ctx context.Context, it *interviewdomain.Interview) error
	GetByID(ctx context.Context, id uuid.UUID) (*interviewdomain.Interview, error)
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]*interviewdomain.Interview, error)
}

type ApplicationStageChecker interface {
	GetStage(ctx context.Context, appID uuid.UUID) (applicationdomain.PipelineStage, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, message eventdomain.Message) error
}

type AuthorizationPort interface {
	CanManageInterview(ctx context.Context, actorID, appID uuid.UUID) error
	CanSubmitFeedback(ctx context.Context, actorID, interviewID uuid.UUID) error
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
	interviewRepo InterviewRepository
	appChecker    ApplicationStageChecker
	roomStrategy  *room.RoomStrategy
	outboxRepo    OutboxRepository
	authPort      AuthorizationPort
	txManager     TransactionManager
	eventFactory  eventdomain.Factory
	clock         Clock
	idGen         IDGenerator
}

func NewService(
	interviewRepo InterviewRepository,
	appChecker ApplicationStageChecker,
	roomStrategy *room.RoomStrategy,
	outboxRepo OutboxRepository,
	authPort AuthorizationPort,
	txManager TransactionManager,
	eventFactory eventdomain.Factory,
	clock Clock,
	idGen IDGenerator,
) *Service {
	return &Service{
		interviewRepo: interviewRepo,
		appChecker:    appChecker,
		roomStrategy:  roomStrategy,
		outboxRepo:    outboxRepo,
		authPort:      authPort,
		txManager:     txManager,
		eventFactory:  eventFactory,
		clock:         clock,
		idGen:         idGen,
	}
}
