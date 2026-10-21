package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/event"
)

const (
	AggregateTypeJobApplication event.AggregateType = "job_application"

	EventTypeApplicationApplied   event.Type = "hiring.application.applied"
	EventTypeStageChanged         event.Type = "hiring.application.stage_changed"
	EventTypeApplicationRejected  event.Type = "hiring.application.rejected"
	EventTypeApplicationWithdrawn event.Type = "hiring.application.withdrawn"
)

type ApplicationAppliedPayload struct {
	ApplicationID uuid.UUID     `json:"application_id"`
	VacancyID     uuid.UUID     `json:"vacancy_id"`
	CandidateID   uuid.UUID     `json:"candidate_id"`
	InitialStage  PipelineStage `json:"initial_stage"`
	AppliedAt     time.Time     `json:"applied_at"`
}

type StageChangedPayload struct {
	ApplicationID uuid.UUID     `json:"application_id"`
	FromStage     PipelineStage `json:"from_stage"`
	ToStage       PipelineStage `json:"to_stage"`
	ChangedBy     uuid.UUID     `json:"changed_by"`
	ChangedAt     time.Time     `json:"changed_at"`
	Reason        string        `json:"reason,omitempty"`
}

type ApplicationRejectedPayload struct {
	ApplicationID uuid.UUID     `json:"application_id"`
	FromStage     PipelineStage `json:"from_stage"`
	RejectReason  RejectReason  `json:"reject_reason"`
	RejectedBy    uuid.UUID     `json:"rejected_by"`
	RejectedAt    time.Time     `json:"rejected_at"`
}

type ApplicationWithdrawnPayload struct {
	ApplicationID uuid.UUID     `json:"application_id"`
	FromStage     PipelineStage `json:"from_stage"`
	WithdrawnBy   uuid.UUID     `json:"withdrawn_by"`
	WithdrawnAt   time.Time     `json:"withdrawn_at"`
	Reason        string        `json:"reason,omitempty"`
}

