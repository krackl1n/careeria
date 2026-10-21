package offer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

type CreateOfferInput struct {
	ApplicationID uuid.UUID
	Compensation  int64
	Currency      string
	Grade         string
	ExpiresAt     time.Time
	ActorID       uuid.UUID
}

func (s *Service) CreateOffer(ctx context.Context, input CreateOfferInput) (uuid.UUID, error) {
	if input.ApplicationID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}
	if input.ActorID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	if err := s.authPort.CanManageOffer(ctx, input.ActorID, input.ApplicationID); err != nil {
		return uuid.Nil, err
	}

	app, err := s.appCoord.GetApplication(ctx, input.ApplicationID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return uuid.Nil, apperr.NotFound("application_not_found", "application not found", err)
		}
		return uuid.Nil, fmt.Errorf("load application: %w", err)
	}

	if app.CurrentStage != applicationdomain.StageOfferPending {
		return uuid.Nil, apperr.Conflict(
			"invalid_stage",
			fmt.Sprintf("cannot create offer for application in stage %s; must be %s",
				app.CurrentStage, applicationdomain.StageOfferPending),
			nil,
		)
	}

	now := s.clock.Now().UTC()
	offerID := s.idGen.New()

	of, err := offerdomain.Create(
		offerID,
		input.ApplicationID,
		input.Compensation,
		input.Currency,
		input.Grade,
		input.ExpiresAt,
		now,
	)
	if err != nil {
		return uuid.Nil, apperr.InvalidArgument("invalid_offer_params", err.Error(), err)
	}

	payload := offerdomain.OfferCreatedPayload{
		OfferID:       offerID,
		ApplicationID: input.ApplicationID,
		Compensation:  input.Compensation,
		Currency:      of.Currency,
		Grade:         of.Grade,
		ExpiresAt:     input.ExpiresAt,
		CreatedAt:     now,
	}

	version := of.Version
	eventMsg, err := s.eventFactory.Create(
		offerdomain.EventTypeOfferCreated,
		offerdomain.AggregateTypeJobOffer,
		offerID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create offer created event: %w", err)
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.offerRepo.Create(txCtx, of); err != nil {
			return fmt.Errorf("save offer: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return offerID, nil
}

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

