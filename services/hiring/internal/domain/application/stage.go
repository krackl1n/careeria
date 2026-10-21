package application

import "fmt"

type PipelineStage string

const (
	StageApplied        PipelineStage = "APPLIED"
	StageScreening      PipelineStage = "SCREENING"
	StageTechAssessment PipelineStage = "TECH_ASSESSMENT"
	StageInterview      PipelineStage = "INTERVIEW"
	StageOfferPending   PipelineStage = "OFFER_PENDING"
	StageOfferAccepted  PipelineStage = "OFFER_ACCEPTED"
	StageHired          PipelineStage = "HIRED"
	StageRejected       PipelineStage = "REJECTED"
	StageWithdrawn      PipelineStage = "WITHDRAWN"
)

func (s PipelineStage) String() string {
	return string(s)
}

func (s PipelineStage) Valid() bool {
	switch s {
	case StageApplied,
		StageScreening,
		StageTechAssessment,
		StageInterview,
		StageOfferPending,
		StageOfferAccepted,
		StageHired,
		StageRejected,
		StageWithdrawn:
		return true
	default:
		return false
	}
}

func (s PipelineStage) IsTerminal() bool {
	switch s {
	case StageHired, StageRejected, StageWithdrawn:
		return true
	default:
		return false
	}
}

func (s PipelineStage) CanTransitionTo(target PipelineStage) bool {
	if !s.Valid() || !target.Valid() {
		return false
	}

	if s == target {
		return false
	}

	if s.IsTerminal() {
		return false
	}

	if target == StageRejected || target == StageWithdrawn {
		return true
	}

	switch s {
	case StageApplied:
		return target == StageScreening
	case StageScreening:
		return target == StageTechAssessment || target == StageInterview
	case StageTechAssessment:
		return target == StageInterview
	case StageInterview:
		return target == StageOfferPending
	case StageOfferPending:
		return target == StageOfferAccepted || target == StageInterview
	case StageOfferAccepted:
		return target == StageHired
	default:
		return false
	}
}

func ParseStage(value string) (PipelineStage, error) {
	stage := PipelineStage(value)
	if !stage.Valid() {
		return "", fmt.Errorf("invalid pipeline stage: %q", value)
	}
	return stage, nil
}

