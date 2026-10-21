package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
)

type PipelineHistoryOutput struct {
	ApplicationID uuid.UUID
	VacancyID     uuid.UUID
	CandidateID   uuid.UUID
	CurrentStage  applicationdomain.PipelineStage
	Version       int64
	History       []applicationdomain.StageTransition
}

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

	if app.CandidateID() != actorID {
		if err := s.authPort.CanManageApplication(ctx, actorID, appID); err != nil {
			return nil, apperr.Forbidden("not_authorized", "actor is not authorized to view application", nil)
		}
	}

	return app, nil
}

func (s *Service) GetPipelineHistory(ctx context.Context, appID, actorID uuid.UUID) (*PipelineHistoryOutput, error) {
	app, err := s.GetApplication(ctx, appID, actorID)
	if err != nil {
		return nil, err
	}

	return &PipelineHistoryOutput{
		ApplicationID: app.ID(),
		VacancyID:     app.VacancyID(),
		CandidateID:   app.CandidateID(),
		CurrentStage:  app.CurrentStage(),
		Version:       app.Version(),
		History:       app.History(),
	}, nil
}

