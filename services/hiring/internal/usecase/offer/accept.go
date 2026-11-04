package offer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

type AcceptOfferInput struct {
	OfferID uuid.UUID
	ActorID uuid.UUID
}

func (s *Service) AcceptOffer(ctx context.Context, input AcceptOfferInput) error {
	if input.OfferID == uuid.Nil {
		return apperr.InvalidArgument("empty_offer_id", "offer ID is required", nil)
	}

	of, err := s.offerRepo.GetByID(ctx, input.OfferID)
	if err != nil {
		if errors.Is(err, offerdomain.ErrNotFound) {
			return apperr.NotFound("offer_not_found", "job offer not found", err)
		}
		return fmt.Errorf("load offer: %w", err)
	}

	if err := s.authPort.CanRespondToOffer(ctx, input.ActorID, of.ApplicationID); err != nil {
		return err
	}

	app, err := s.appRepo.GetByID(ctx, of.ApplicationID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return apperr.NotFound("application_not_found", "application not found", err)
		}
		return fmt.Errorf("load application: %w", err)
	}

	now := s.clock.Now().UTC()
	if err := of.Accept(now); err != nil {
		if errors.Is(err, offerdomain.ErrOfferExpired) {
			return apperr.Conflict("offer_expired", err.Error(), err)
		}
		if errors.Is(err, offerdomain.ErrInvalidStatus) {
			return apperr.Conflict("invalid_status", err.Error(), err)
		}
		return err
	}

	if err := app.ChangeStage(applicationdomain.StageOfferAccepted, input.ActorID, "offer accepted by candidate", now); err != nil {
		return apperr.Conflict("invalid_stage_transition", err.Error(), err)
	}

	payload := offerdomain.OfferAcceptedPayload{
		OfferID:    of.ID,
		AcceptedAt: now,
	}

	version := of.Version
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferAccepted,
		offerdomain.AggregateTypeJobOffer,
		of.ID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create offer accepted event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.offerRepo.Update(txCtx, of); err != nil {
			return fmt.Errorf("update offer: %w", err)
		}
		if err := s.appRepo.Update(txCtx, app); err != nil {
			return fmt.Errorf("update application: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}
