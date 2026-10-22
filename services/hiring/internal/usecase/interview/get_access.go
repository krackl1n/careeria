package interview

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/krackl1n/careeria/services/hiring/internal/usecase/interview/room"
)

type GetMeetingAccessInput struct {
	InterviewID uuid.UUID
	ActorID     uuid.UUID
}

type MeetingAccessOutput struct {
	InterviewID  uuid.UUID
	Format       interviewdomain.MeetingFormat
	JoinURL      string
	Token        string
	Instructions string
}

func (s *Service) GetMeetingAccess(ctx context.Context, input GetMeetingAccessInput) (*MeetingAccessOutput, error) {
	if input.InterviewID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_interview_id", "interview ID is required", nil)
	}
	if input.ActorID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	it, err := s.interviewRepo.GetByID(ctx, input.InterviewID)
	if err != nil {
		if errors.Is(err, interviewdomain.ErrNotFound) {
			return nil, apperr.NotFound("interview_not_found", "interview not found", err)
		}
		return nil, fmt.Errorf("load interview: %w", err)
	}

	if it.Status != interviewdomain.StatusScheduled {
		return nil, apperr.Conflict("invalid_status", "interview is not in scheduled status", nil)
	}

	userRole := interviewdomain.RoleCandidate
	for _, p := range it.Participants {
		if p.UserID == input.ActorID {
			userRole = p.Role
			break
		}
	}

	if s.roomStrategy == nil {
		return &MeetingAccessOutput{
			InterviewID: it.ID,
			Format:      it.Format,
			JoinURL:     it.MeetingRoomURL,
		}, nil
	}

	res, err := s.roomStrategy.GenerateAccess(ctx, it.Format, room.AccessRequest{
		InterviewID: it.ID,
		UserID:      input.ActorID,
		UserRole:    userRole,
		MeetingURL:  it.MeetingRoomURL,
	})
	if err != nil {
		if errors.Is(err, room.ErrUnsupportedFormat) {
			return nil, apperr.Conflict("unsupported_format", err.Error(), err)
		}
		return nil, fmt.Errorf("generate access: %w", err)
	}

	return &MeetingAccessOutput{
		InterviewID:  it.ID,
		Format:       it.Format,
		JoinURL:      res.JoinURL,
		Token:        res.Token,
		Instructions: res.Instructions,
	}, nil
}
