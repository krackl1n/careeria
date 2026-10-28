package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

var ErrUnsupportedFormat = errors.New("unsupported meeting format")

type ProvisionRequest struct {
	InterviewID   uuid.UUID
	ApplicationID uuid.UUID
	InterviewType interviewdomain.InterviewType
	StartTime     time.Time
	EndTime       time.Time
	Location      string
}

type ProvisionResult struct {
	MeetingURL string
	Metadata   map[string]string
}

type AccessRequest struct {
	InterviewID uuid.UUID
	UserID      uuid.UUID
	UserRole    interviewdomain.ParticipantRole
	MeetingURL  string
}

type AccessResult struct {
	JoinURL      string
	Token        string
	Instructions string
}

type TeardownRequest struct {
	InterviewID uuid.UUID
	MeetingURL  string
}

type MeetingRoomStrategy interface {
	Format() interviewdomain.MeetingFormat
	Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
	GenerateAccess(ctx context.Context, req AccessRequest) (*AccessResult, error)
	Teardown(ctx context.Context, req TeardownRequest) error
}

type RoomStrategy struct {
	registry map[interviewdomain.MeetingFormat]MeetingRoomStrategy
}

func NewRoomStrategy(strategies ...MeetingRoomStrategy) *RoomStrategy {
	registry := make(map[interviewdomain.MeetingFormat]MeetingRoomStrategy, len(strategies))
	for _, s := range strategies {
		if s != nil {
			registry[s.Format()] = s
		}
	}
	return &RoomStrategy{registry: registry}
}

func (s *RoomStrategy) Resolve(format interviewdomain.MeetingFormat) (MeetingRoomStrategy, error) {
	if s == nil || s.registry == nil {
		return nil, ErrUnsupportedFormat
	}
	strat, ok := s.registry[format]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return strat, nil
}

func (s *RoomStrategy) Provision(ctx context.Context, format interviewdomain.MeetingFormat, req ProvisionRequest) (*ProvisionResult, error) {
	strat, err := s.Resolve(format)
	if err != nil {
		return nil, err
	}
	return strat.Provision(ctx, req)
}

func (s *RoomStrategy) GenerateAccess(ctx context.Context, format interviewdomain.MeetingFormat, req AccessRequest) (*AccessResult, error) {
	strat, err := s.Resolve(format)
	if err != nil {
		return nil, err
	}
	return strat.GenerateAccess(ctx, req)
}

func (s *RoomStrategy) Teardown(ctx context.Context, format interviewdomain.MeetingFormat, req TeardownRequest) error {
	strat, err := s.Resolve(format)
	if err != nil {
		return err
	}
	return strat.Teardown(ctx, req)
}
