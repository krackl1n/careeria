package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
)

type AssessmentCompletedInput struct {
	ApplicationID uuid.UUID
	Score         int
	Passed        bool
	ActorID       uuid.UUID
}

func (s *Service) HandleAssessmentResult(ctx context.Context, input AssessmentCompletedInput) error {
	if input.ApplicationID == uuid.Nil {
		return apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}

	app, err := s.appRepo.GetByID(ctx, input.ApplicationID)
	if err != nil {
		return fmt.Errorf("load application: %w", err)
	}

	if app.CurrentStage != applicationdomain.StageTechAssessment {
		return nil
	}

	now := s.clock.Now().UTC()
	fromStage := app.CurrentStage

	if input.Passed {
		if err := app.AdvanceFromAssessment(input.Score, input.ActorID, now); err != nil {
			if errors.Is(err, applicationdomain.ErrInvalidStageTransition) {
				return apperr.Conflict("invalid_stage_transition", err.Error(), err)
			}
			return err
		}

		payload := applicationdomain.StageChangedPayload{
			ApplicationID: app.ID,
			FromStage:     fromStage,
			ToStage:       applicationdomain.StageInterview,
			ChangedBy:     input.ActorID,
			ChangedAt:     now,
			Reason:        fmt.Sprintf("tech assessment passed with score %d", input.Score),
		}

		version := app.Version
		eventMsg, err := s.eventFactory.Create(
			applicationdomain.EventTypeStageChanged,
			applicationdomain.AggregateTypeJobApplication,
			app.ID,
			&version,
			payload,
			now,
		)
		if err != nil {
			return fmt.Errorf("create stage changed event: %w", err)
		}

		return s.txManager.Do(ctx, func(txCtx context.Context) error {
			if err := s.appRepo.Update(txCtx, app); err != nil {
				return fmt.Errorf("update application: %w", err)
			}
			if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
				return fmt.Errorf("save outbox event: %w", err)
			}
			return nil
		})
	}

	reason, err := applicationdomain.NewRejectReason(
		applicationdomain.RejectCodeAssessmentFailed,
		fmt.Sprintf("score %d below passing threshold", input.Score),
	)
	if err != nil {
		return err
	}
	if err := app.Reject(reason, input.ActorID, now); err != nil {
		if errors.Is(err, applicationdomain.ErrTerminalStage) {
			return apperr.Conflict("terminal_stage", err.Error(), err)
		}
		if errors.Is(err, applicationdomain.ErrInvalidStageTransition) {
			return apperr.Conflict("invalid_stage_transition", err.Error(), err)
		}
		return err
	}

	payload := applicationdomain.ApplicationRejectedPayload{
		ApplicationID: app.ID,
		FromStage:     fromStage,
		RejectReason:  reason,
		RejectedBy:    input.ActorID,
		RejectedAt:    now,
	}

	version := app.Version
	eventMsg, err := s.eventFactory.Create(
		applicationdomain.EventTypeApplicationRejected,
		applicationdomain.AggregateTypeJobApplication,
		app.ID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create rejected event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.appRepo.Update(txCtx, app); err != nil {
			return fmt.Errorf("update application: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}

