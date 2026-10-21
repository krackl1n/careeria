package interview

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

func (s *Service) GetInterview(ctx context.Context, interviewID, actorID uuid.UUID) (*interviewdomain.Interview, error) {
	if interviewID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_interview_id", "interview ID is required", nil)
	}

	it, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, interviewdomain.ErrNotFound) {
			return nil, apperr.NotFound("interview_not_found", "interview not found", err)
		}
		return nil, fmt.Errorf("load interview: %w", err)
	}

	if err := s.authPort.CanManageInterview(ctx, actorID, it.ApplicationID()); err != nil {
		return nil, apperr.Forbidden("not_authorized", "actor is not authorized to view interview", nil)
	}

	return it, nil
}

func (s *Service) ListInterviewsByApplication(ctx context.Context, appID, actorID uuid.UUID) ([]*interviewdomain.Interview, error) {
	if appID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}

	if err := s.authPort.CanManageInterview(ctx, actorID, appID); err != nil {
		return nil, apperr.Forbidden("not_authorized", "actor is not authorized to list interviews", nil)
	}

	return s.interviewRepo.ListByApplicationID(ctx, appID)
}

