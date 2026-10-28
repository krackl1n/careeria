package interview

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type CompleteInterviewInput struct {
	InterviewID uuid.UUID
	ActorID     uuid.UUID
}

func (s *Service) CompleteInterview(ctx context.Context, input CompleteInterviewInput) error {
	if input.InterviewID == uuid.Nil {
		return apperr.InvalidArgument("empty_interview_id", "interview ID is required", nil)
	}

	it, err := s.interviewRepo.GetByID(ctx, input.InterviewID)
	if err != nil {
		if errors.Is(err, interviewdomain.ErrNotFound) {
			return apperr.NotFound("interview_not_found", "interview not found", err)
		}
		return fmt.Errorf("load interview: %w", err)
	}

	if err := s.authPort.CanManageInterview(ctx, input.ActorID, it.ApplicationID); err != nil {
		return err
	}

	now := s.clock.Now().UTC()
	if err := it.Complete(now); err != nil {
		if errors.Is(err, interviewdomain.ErrInvalidStatus) {
			return apperr.Conflict("invalid_status", err.Error(), err)
		}
		return err
	}

	payload := interviewdomain.InterviewCompletedPayload{
		InterviewID: it.ID,
		CompletedAt: now,
	}

	version := it.Version
	eventMsg, err := s.eventFactory.Create(
		interviewdomain.EventTypeInterviewCompleted,
		interviewdomain.AggregateTypeInterview,
		it.ID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create interview completed event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.interviewRepo.Update(txCtx, it); err != nil {
			return fmt.Errorf("update interview: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}
