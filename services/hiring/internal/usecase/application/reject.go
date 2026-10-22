package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
)

type RejectInput struct {
	ApplicationID uuid.UUID
	ReasonCode    applicationdomain.RejectCode
	Comment       string
	ActorID       uuid.UUID
}

func (s *Service) RejectApplication(ctx context.Context, input RejectInput) error {
	if input.ApplicationID == uuid.Nil {
		return apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
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
	now := s.clock.Now().UTC()

	reason, err := applicationdomain.NewRejectReason(input.ReasonCode, input.Comment)
	if err != nil {
		return apperr.InvalidArgument("invalid_reject_reason", err.Error(), err)
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
