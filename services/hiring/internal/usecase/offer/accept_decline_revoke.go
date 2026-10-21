package offer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
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

	if err := s.authPort.CanRespondToOffer(ctx, input.ActorID, of.ApplicationID()); err != nil {
		return err
	}

	now := s.clock.Now().UTC()
	if err := of.Accept(now); err != nil {
		return err
	}

	payload := offerdomain.OfferAcceptedPayload{
		OfferID:    of.ID(),
		AcceptedAt: now,
	}

	version := of.Version()
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferAccepted,
		offerdomain.AggregateTypeJobOffer,
		of.ID(),
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
		if err := s.appCoord.AdvanceToOfferAccepted(txCtx, of.ApplicationID(), input.ActorID, now); err != nil {
			return fmt.Errorf("coordinate application stage: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}

type DeclineOfferInput struct {
	OfferID uuid.UUID
	Reason  string
	ActorID uuid.UUID
}

func (s *Service) DeclineOffer(ctx context.Context, input DeclineOfferInput) error {
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

	if err := s.authPort.CanRespondToOffer(ctx, input.ActorID, of.ApplicationID()); err != nil {
		return err
	}

	now := s.clock.Now().UTC()
	if err := of.Decline(input.Reason, now); err != nil {
		return err
	}

	payload := offerdomain.OfferDeclinedPayload{
		OfferID:    of.ID(),
		Reason:     input.Reason,
		DeclinedAt: now,
	}

	version := of.Version()
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferDeclined,
		offerdomain.AggregateTypeJobOffer,
		of.ID(),
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create offer declined event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.offerRepo.Update(txCtx, of); err != nil {
			return fmt.Errorf("update offer: %w", err)
		}
		if err := s.appCoord.RevertToInterview(txCtx, of.ApplicationID(), input.ActorID, "offer declined by candidate", now); err != nil {
			return fmt.Errorf("coordinate application stage: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}

type RevokeOfferInput struct {
	OfferID uuid.UUID
	Reason  string
	ActorID uuid.UUID
}

func (s *Service) RevokeOffer(ctx context.Context, input RevokeOfferInput) error {
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

	if err := s.authPort.CanManageOffer(ctx, input.ActorID, of.ApplicationID()); err != nil {
		return err
	}

	now := s.clock.Now().UTC()
	if err := of.Revoke(input.Reason, now); err != nil {
		return err
	}

	payload := offerdomain.OfferRevokedPayload{
		OfferID:   of.ID(),
		Reason:    input.Reason,
		RevokedAt: now,
	}

	version := of.Version()
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferRevoked,
		offerdomain.AggregateTypeJobOffer,
		of.ID(),
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create offer revoked event: %w", err)
	}

	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.offerRepo.Update(txCtx, of); err != nil {
			return fmt.Errorf("update offer: %w", err)
		}
		if err := s.appCoord.RevertToInterview(txCtx, of.ApplicationID(), input.ActorID, "offer revoked", now); err != nil {
			return fmt.Errorf("coordinate application stage: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
}

func (s *Service) ExpireOffer(ctx context.Context, offerID uuid.UUID) error {
	if offerID == uuid.Nil {
		return apperr.InvalidArgument("empty_offer_id", "offer ID is required", nil)
	}

	of, err := s.offerRepo.GetByID(ctx, offerID)
	if err != nil {
		if errors.Is(err, offerdomain.ErrNotFound) {
			return apperr.NotFound("offer_not_found", "job offer not found", err)
		}
		return fmt.Errorf("load offer: %w", err)
	}

	now := s.clock.Now().UTC()
	if err := of.Expire(now); err != nil {
		return err
	}

	payload := offerdomain.OfferExpiredPayload{
		OfferID:   of.ID(),
		ExpiredAt: now,
	}

	version := of.Version()
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferExpired,
		offerdomain.AggregateTypeJobOffer,
		of.ID(),
		&version,
		payload,
		now,
	)
	if err != nil {
		return fmt.Errorf("create offer expired event: %w", err)
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

