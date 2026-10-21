package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JobApplication struct {
	id           uuid.UUID
	vacancyID    uuid.UUID
	candidateID  uuid.UUID
	currentStage PipelineStage
	version      int64
	history      []StageTransition
	createdAt    time.Time
	updatedAt    time.Time
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
		id:           id,
		vacancyID:    vacancyID,
		candidateID:  candidateID,
		currentStage: StageApplied,
		version:      1,
		createdAt:    now,
		updatedAt:    now,
		history: []StageTransition{
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
		id:           id,
		vacancyID:    vacancyID,
		candidateID:  candidateID,
		currentStage: currentStage,
		version:      version,
		history:      history,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (a *JobApplication) ID() uuid.UUID {
	return a.id
}

func (a *JobApplication) VacancyID() uuid.UUID {
	return a.vacancyID
}

func (a *JobApplication) CandidateID() uuid.UUID {
	return a.candidateID
}

func (a *JobApplication) CurrentStage() PipelineStage {
	return a.currentStage
}

func (a *JobApplication) Version() int64 {
	return a.version
}

func (a *JobApplication) History() []StageTransition {
	copied := make([]StageTransition, len(a.history))
	copy(copied, a.history)
	return copied
}

func (a *JobApplication) CreatedAt() time.Time {
	return a.createdAt
}

func (a *JobApplication) UpdatedAt() time.Time {
	return a.updatedAt
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

	if !a.currentStage.CanTransitionTo(targetStage) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStageTransition, a.currentStage, targetStage)
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
	if a.currentStage != StageTechAssessment {
		return fmt.Errorf("%w: cannot advance from assessment while in stage %s", ErrInvalidStageTransition, a.currentStage)
	}

	reason := fmt.Sprintf("assessment passed with score %d", score)
	return a.ChangeStage(StageInterview, actorID, reason, now)
}

func (a *JobApplication) Reject(reason RejectReason, actorID uuid.UUID, now time.Time) error {
	if a.currentStage.IsTerminal() {
		return fmt.Errorf("%w: cannot reject application in stage %s", ErrTerminalStage, a.currentStage)
	}

	comment := reason.Comment
	if comment == "" && reason.Code != "" {
		comment = reason.Code
	}

	return a.ChangeStage(StageRejected, actorID, comment, now)
}

func (a *JobApplication) Withdraw(actorID uuid.UUID, reason string, now time.Time) error {
	if a.currentStage.IsTerminal() {
		return fmt.Errorf("%w: cannot withdraw application in stage %s", ErrTerminalStage, a.currentStage)
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
		FromStage: a.currentStage,
		ToStage:   to,
		ChangedBy: by,
		ChangedAt: now,
		Reason:    reason,
	}

	a.history = append(a.history, transition)
	a.currentStage = to
	a.version++
	a.updatedAt = now
}
