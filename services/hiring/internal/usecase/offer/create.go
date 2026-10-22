package offer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

type CreateOfferInput struct {
	ApplicationID uuid.UUID
	Compensation  int64
	Currency      offerdomain.Currency
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

	app, err := s.appRepo.GetByID(ctx, input.ApplicationID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return uuid.Nil, apperr.NotFound("application_not_found", "application not found", err)
		}
		return uuid.Nil, fmt.Errorf("load application: %w", err)
	}

	now := s.clock.Now().UTC()

	needStageUpdate := false
	if app.CurrentStage == applicationdomain.StageInterview {
		if err := app.ChangeStage(applicationdomain.StageOfferPending, input.ActorID, "offer created", now); err != nil {
			return uuid.Nil, apperr.Conflict("invalid_stage_transition", err.Error(), err)
		}
		needStageUpdate = true
	} else if app.CurrentStage != applicationdomain.StageOfferPending {
		return uuid.Nil, apperr.Conflict(
			"invalid_stage",
			fmt.Sprintf("cannot create offer for application in stage %s; must be %s or %s",
				app.CurrentStage, applicationdomain.StageInterview, applicationdomain.StageOfferPending),
			nil,
		)
	}

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
		if needStageUpdate {
			if err := s.appRepo.Update(txCtx, app); err != nil {
				return fmt.Errorf("update application: %w", err)
			}
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
