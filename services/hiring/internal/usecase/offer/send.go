package offer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

type SendOfferInput struct {
	OfferID uuid.UUID
	ActorID uuid.UUID
}

func (s *Service) SendOffer(ctx context.Context, input SendOfferInput) error {
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

	if err := s.authPort.CanManageOffer(ctx, input.ActorID, of.ApplicationID); err != nil {
		return err
	}

	now := s.clock.Now().UTC()
	if err := of.Send(now); err != nil {
		if errors.Is(err, offerdomain.ErrOfferExpired) {
			return apperr.Conflict("offer_expired", err.Error(), err)
		}
		if errors.Is(err, offerdomain.ErrInvalidStatus) {
			return apperr.Conflict("invalid_status", err.Error(), err)
		}
		return err
	}

	payload := offerdomain.OfferSentPayload{
		OfferID:   of.ID,
		SentAt:    now,
		ExpiresAt: of.ExpiresAt,
	}

	version := of.Version
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferSent,
		offerdomain.AggregateTypeJobOffer,
		of.ID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create offer sent event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.offerRepo.Update(txCtx, of); err != nil {
			return fmt.Errorf("update offer: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}

