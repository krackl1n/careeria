package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
)

type WithdrawInput struct {
	ApplicationID uuid.UUID
	Reason        string
	ActorID       uuid.UUID
}

func (s *Service) WithdrawApplication(ctx context.Context, input WithdrawInput) error {
	if input.ApplicationID == uuid.Nil {
		return apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}
	if input.ActorID == uuid.Nil {
		return apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	app, err := s.appRepo.GetByID(ctx, input.ApplicationID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return apperr.NotFound("application_not_found", "application not found", err)
		}
		return fmt.Errorf("load application: %w", err)
	}

	if app.CandidateID != input.ActorID {
		if err := s.authPort.CanManageApplication(ctx, input.ActorID, input.ApplicationID); err != nil {
			return apperr.Forbidden("not_authorized", "actor is not authorized to withdraw application", nil)
		}
	}

	fromStage := app.CurrentStage
	now := s.clock.Now().UTC()

	if err := app.Withdraw(input.ActorID, input.Reason, now); err != nil {
		if errors.Is(err, applicationdomain.ErrTerminalStage) {
			return apperr.Conflict("terminal_stage", err.Error(), err)
		}
		if errors.Is(err, applicationdomain.ErrInvalidStageTransition) {
			return apperr.Conflict("invalid_stage_transition", err.Error(), err)
		}
		return err
	}

	payload := applicationdomain.ApplicationWithdrawnPayload{
		ApplicationID: app.ID,
		FromStage:     fromStage,
		WithdrawnBy:   input.ActorID,
		WithdrawnAt:   now,
		Reason:        input.Reason,
	}

	version := app.Version
	eventMsg, err := s.eventFactory.Create(
		applicationdomain.EventTypeApplicationWithdrawn,
		applicationdomain.AggregateTypeJobApplication,
		app.ID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create withdrawn event: %w", err)
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
