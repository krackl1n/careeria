package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
)

func (s *Service) GetApplication(ctx context.Context, appID, actorID uuid.UUID) (*applicationdomain.JobApplication, error) {
	if appID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}

	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		if errors.Is(err, applicationdomain.ErrNotFound) {
			return nil, apperr.NotFound("application_not_found", "application not found", err)
		}
		return nil, fmt.Errorf("load application: %w", err)
	}

	if app.CandidateID != actorID {
		if err := s.authPort.CanManageApplication(ctx, actorID, appID); err != nil {
			return nil, apperr.Forbidden("not_authorized", "actor is not authorized to view application", nil)
		}
	}

	return app, nil
}
