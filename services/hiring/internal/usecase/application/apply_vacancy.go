package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
)

type ApplyVacancyInput struct {
	VacancyID   uuid.UUID
	CandidateID uuid.UUID
	ActorID     uuid.UUID
}

func (s *Service) ApplyVacancy(ctx context.Context, input ApplyVacancyInput) (uuid.UUID, error) {
	if input.VacancyID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_vacancy_id", "vacancy ID is required", nil)
	}
	if input.CandidateID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_candidate_id", "candidate ID is required", nil)
	}
	if input.ActorID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	if err := s.authPort.CanApplyVacancy(ctx, input.ActorID, input.VacancyID); err != nil {
		return uuid.Nil, err
	}

	existing, err := s.appRepo.GetActiveByCandidateAndVacancy(ctx, input.CandidateID, input.VacancyID)
	if err != nil && !errors.Is(err, applicationdomain.ErrNotFound) {
		return uuid.Nil, fmt.Errorf("check existing application: %w", err)
	}
	if existing != nil {
		return uuid.Nil, apperr.Conflict("active_application_exists", "candidate already has an active application for this vacancy", nil)
	}

	appID := s.idGen.New()
	now := s.clock.Now().UTC()

	app, err := applicationdomain.New(appID, input.VacancyID, input.CandidateID, input.ActorID, now)
	if err != nil {
		return uuid.Nil, err
	}

	payload := applicationdomain.ApplicationAppliedPayload{
		ApplicationID: appID,
		VacancyID:     input.VacancyID,
		CandidateID:   input.CandidateID,
		InitialStage:  app.CurrentStage(),
		AppliedAt:     now,
	}

	version := app.Version()
	eventMsg, err := s.eventFactory.Create(
		applicationdomain.EventTypeApplicationApplied,
		applicationdomain.AggregateTypeJobApplication,
		appID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create applied event: %w", err)
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.appRepo.Create(txCtx, app); err != nil {
			return fmt.Errorf("save application: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return appID, nil
}

