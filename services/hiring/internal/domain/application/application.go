package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JobApplication struct {
	ID           uuid.UUID
	VacancyID    uuid.UUID
	CandidateID  uuid.UUID
	CurrentStage PipelineStage
	Version      int64
	History      []StageTransition
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func New(
	id uuid.UUID,
	vacancyID uuid.UUID,
	candidateID uuid.UUID,
	actorID uuid.UUID,
	now time.Time,
) (*JobApplication, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if vacancyID == uuid.Nil {
		return nil, ErrEmptyVacancyID
	}
	if candidateID == uuid.Nil {
		return nil, ErrEmptyCandidateID
	}
	if actorID == uuid.Nil {
		return nil, ErrEmptyActorID
	}

	app := &JobApplication{
		ID:           id,
		VacancyID:    vacancyID,
		CandidateID:  candidateID,
		CurrentStage: StageApplied,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
		History: []StageTransition{
			{
				ID:        uuid.New(),
				FromStage: "",
				ToStage:   StageApplied,
				ChangedBy: actorID,
				ChangedAt: now,
				Reason:    "application submitted",
			},
		},
	}

	return app, nil
}

func Reconstitute(
	id uuid.UUID,
	vacancyID uuid.UUID,
	candidateID uuid.UUID,
	currentStage PipelineStage,
	version int64,
	history []StageTransition,
	createdAt time.Time,
	updatedAt time.Time,
) *JobApplication {
	return &JobApplication{
		ID:           id,
		VacancyID:    vacancyID,
		CandidateID:  candidateID,
		CurrentStage: currentStage,
		Version:      version,
		History:      history,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func (a *JobApplication) ChangeStage(
	targetStage PipelineStage,
	actorID uuid.UUID,
	reason string,
	now time.Time,
) error {
	if actorID == uuid.Nil {
		return ErrEmptyActorID
	}

	if !a.CurrentStage.CanTransitionTo(targetStage) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStageTransition, a.CurrentStage, targetStage)
	}

	a.addTransition(targetStage, actorID, reason, now)
	return nil
}

func (a *JobApplication) MarkInterview(actorID uuid.UUID, comment string, now time.Time) error {
	return a.ChangeStage(StageInterview, actorID, comment, now)
}

func (a *JobApplication) MarkOfferPending(actorID uuid.UUID, now time.Time) error {
	return a.ChangeStage(StageOfferPending, actorID, "offer initiated", now)
}

func (a *JobApplication) AdvanceFromAssessment(score int, actorID uuid.UUID, now time.Time) error {
	if a.CurrentStage != StageTechAssessment {
		return fmt.Errorf("%w: cannot advance from assessment while in stage %s", ErrInvalidStageTransition, a.CurrentStage)
	}

	reason := fmt.Sprintf("assessment passed with score %d", score)
	return a.ChangeStage(StageInterview, actorID, reason, now)
}

func (a *JobApplication) Reject(reason RejectReason, actorID uuid.UUID, now time.Time) error {
	if a.CurrentStage.IsTerminal() {
		return fmt.Errorf("%w: cannot reject application in stage %s", ErrTerminalStage, a.CurrentStage)
	}

	comment := reason.Comment
	if comment == "" && reason.Code != "" {
		comment = reason.Code
	}

	return a.ChangeStage(StageRejected, actorID, comment, now)
}

func (a *JobApplication) Withdraw(actorID uuid.UUID, reason string, now time.Time) error {
	if a.CurrentStage.IsTerminal() {
		return fmt.Errorf("%w: cannot withdraw application in stage %s", ErrTerminalStage, a.CurrentStage)
	}

	if reason == "" {
		reason = "candidate withdrawn"
	}

	return a.ChangeStage(StageWithdrawn, actorID, reason, now)
}

func (a *JobApplication) addTransition(
	to PipelineStage,
	by uuid.UUID,
	reason string,
	now time.Time,
) {
	transition := StageTransition{
		ID:        uuid.New(),
		FromStage: a.CurrentStage,
		ToStage:   to,
		ChangedBy: by,
		ChangedAt: now,
		Reason:    reason,
	}

	a.History = append(a.History, transition)
	a.CurrentStage = to
	a.Version++
	a.UpdatedAt = now
}
