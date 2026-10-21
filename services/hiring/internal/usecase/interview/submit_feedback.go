package interview

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type SubmitFeedbackInput struct {
	InterviewID       uuid.UUID
	InterviewerUserID uuid.UUID
	Score             int
	Notes             string
}

func (s *Service) SubmitFeedback(ctx context.Context, input SubmitFeedbackInput) error {
	if input.InterviewID == uuid.Nil {
		return apperr.InvalidArgument("empty_interview_id", "interview ID is required", nil)
	}
	if input.InterviewerUserID == uuid.Nil {
		return apperr.InvalidArgument("empty_interviewer_id", "interviewer user ID is required", nil)
	}

	if err := s.authPort.CanSubmitFeedback(ctx, input.InterviewerUserID, input.InterviewID); err != nil {
		return err
	}

	it, err := s.interviewRepo.GetByID(ctx, input.InterviewID)
	if err != nil {
		if errors.Is(err, interviewdomain.ErrNotFound) {
			return apperr.NotFound("interview_not_found", "interview not found", err)
		}
		return fmt.Errorf("load interview: %w", err)
	}

	now := s.clock.Now().UTC()
	feedback := interviewdomain.Feedback{
		InterviewerUserID: input.InterviewerUserID,
		Score:             input.Score,
		Notes:             input.Notes,
		SubmittedAt:       now,
	}

	if err := it.SubmitFeedback(feedback, now); err != nil {
		if errors.Is(err, interviewdomain.ErrNotParticipant) {
			return apperr.Forbidden("not_participant", err.Error(), err)
		}
		if errors.Is(err, interviewdomain.ErrFeedbackAlreadySubmitted) {
			return apperr.Conflict("feedback_exists", err.Error(), err)
		}
		if errors.Is(err, interviewdomain.ErrInterviewCancelled) {
			return apperr.Conflict("interview_cancelled", err.Error(), err)
		}
		return apperr.InvalidArgument("invalid_feedback", err.Error(), err)
	}

	payload := interviewdomain.FeedbackSubmittedPayload{
		InterviewID:       it.ID(),
		InterviewerUserID: input.InterviewerUserID,
		Score:             input.Score,
		SubmittedAt:       now,
	}

	version := it.Version()
	eventMsg, err := s.eventFactory.Create(
		interviewdomain.EventTypeFeedbackSubmitted,
		interviewdomain.AggregateTypeInterview,
		it.ID(),
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create feedback submitted event: %w", err)
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

