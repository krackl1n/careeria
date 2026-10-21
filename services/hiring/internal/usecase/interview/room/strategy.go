package room

import (
	"context"
	"time"

	"github.com/google/uuid"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type ProvisionRequest struct {
	InterviewID   uuid.UUID
	ApplicationID uuid.UUID
	InterviewType string
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
	UserRole    string
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

