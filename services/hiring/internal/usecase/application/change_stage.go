package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	policydomain "github.com/krackl1n/careeria/services/hiring/internal/domain/policy"
)

type ChangeStageInput struct {
	ApplicationID uuid.UUID
	TargetStage   applicationdomain.PipelineStage
	ActorID       uuid.UUID
	Reason        string
}

func (s *Service) ChangeStage(ctx context.Context, input ChangeStageInput) error {
	if input.ApplicationID == uuid.Nil {
		return apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}
	if !input.TargetStage.Valid() {
		return apperr.InvalidArgument("invalid_stage", "target stage is invalid", nil)
	}
	if input.ActorID == uuid.Nil {
		return apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	if err := s.authPort.CanManageApplication(ctx, input.ActorID, input.ApplicationID); err != nil {
		return err
	}

	app, err := s.appRepo.GetByID(ctx, input.ApplicationID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return apperr.NotFound("application_not_found", "application not found", err)
		}
		return fmt.Errorf("load application: %w", err)
	}

	fromStage := app.CurrentStage

	if input.TargetStage == applicationdomain.StageOfferPending {
		interviews, err := s.interviewRepo.ListByApplicationID(ctx, input.ApplicationID)
		if err != nil {
			return fmt.Errorf("fetch interviews: %w", err)
		}

		if err := s.offerPolicy.CanInitiateOffer(app, interviews); err != nil {
			if errors.Is(err, policydomain.ErrInvalidStageForOffer) {
				return apperr.Conflict("invalid_stage_for_offer", err.Error(), err)
			}
			if errors.Is(err, policydomain.ErrMissingCompletedInterview) {
				return apperr.Conflict("missing_completed_interview", err.Error(), err)
			}
			return apperr.InvalidArgument("offer_eligibility_failed", err.Error(), err)
		}
	}

	now := s.clock.Now().UTC()
	if err := app.ChangeStage(input.TargetStage, input.ActorID, input.Reason, now); err != nil {
		if errors.Is(err, applicationdomain.ErrInvalidStageTransition) {
			return apperr.Conflict("invalid_stage_transition", err.Error(), err)
		}
		if errors.Is(err, applicationdomain.ErrEmptyActorID) {
			return apperr.InvalidArgument("empty_actor_id", err.Error(), err)
		}
		return err
	}

	payload := applicationdomain.StageChangedPayload{
		ApplicationID: app.ID,
		FromStage:     fromStage,
		ToStage:       input.TargetStage,
		ChangedBy:     input.ActorID,
		ChangedAt:     now,
		Reason:        input.Reason,
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

