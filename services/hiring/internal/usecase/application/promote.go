package application

import (
	"context"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
)

func (s *Service) PromoteToInterview(ctx context.Context, appID, actorID uuid.UUID, reason string) error {
	return s.ChangeStage(ctx, ChangeStageInput{
		ApplicationID: appID,
		TargetStage:   applicationdomain.StageInterview,
		ActorID:       actorID,
		Reason:        reason,
	})
}
