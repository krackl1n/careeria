package application

import (
	"context"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
)

type PipelineHistoryOutput struct {
	ApplicationID uuid.UUID
	VacancyID     uuid.UUID
	CandidateID   uuid.UUID
	CurrentStage  applicationdomain.PipelineStage
	Version       int64
	History       []applicationdomain.StageTransition
}

func (s *Service) GetPipelineHistory(ctx context.Context, appID, actorID uuid.UUID) (*PipelineHistoryOutput, error) {
	app, err := s.GetApplication(ctx, appID, actorID)
	if err != nil {
		return nil, err
	}

	return &PipelineHistoryOutput{
		ApplicationID: app.ID,
		VacancyID:     app.VacancyID,
		CandidateID:   app.CandidateID,
		CurrentStage:  app.CurrentStage,
		Version:       app.Version,
		History:       app.History,
	}, nil
}

