package offer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
)

func (s *Service) GetOffer(ctx context.Context, offerID, actorID uuid.UUID) (*offerdomain.JobOffer, error) {
	if offerID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_offer_id", "offer ID is required", nil)
	}

	of, err := s.offerRepo.GetByID(ctx, offerID)
	if err != nil {
		if errors.Is(err, offerdomain.ErrNotFound) {
			return nil, apperr.NotFound("offer_not_found", "job offer not found", err)
		}
		return nil, fmt.Errorf("load offer: %w", err)
	}

	if err := s.authPort.CanManageOffer(ctx, actorID, of.ApplicationID); err != nil {
		if errResp := s.authPort.CanRespondToOffer(ctx, actorID, of.ApplicationID); errResp != nil {
			return nil, apperr.Forbidden("not_authorized", "actor is not authorized to view offer", nil)
		}
	}

	return of, nil
}

func (s *Service) GetOfferByApplication(ctx context.Context, appID, actorID uuid.UUID) (*offerdomain.JobOffer, error) {
	if appID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}

	if err := s.authPort.CanManageOffer(ctx, actorID, appID); err != nil {
		if errResp := s.authPort.CanRespondToOffer(ctx, actorID, appID); errResp != nil {
			return nil, apperr.Forbidden("not_authorized", "actor is not authorized to view offer", nil)
		}
	}

	return s.offerRepo.GetByApplicationID(ctx, appID)
}

