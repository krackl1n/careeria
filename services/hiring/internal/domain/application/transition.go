package application

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTransitionActor = errors.New("transition actor ID must not be empty")
	ErrInvalidTransitionStage = errors.New("transition target stage is invalid")
	ErrEmptyRejectReason       = errors.New("reject reason code or comment is required")
)

type StageTransition struct {
	ID        uuid.UUID
	FromStage PipelineStage
	ToStage   PipelineStage
	ChangedBy uuid.UUID
	ChangedAt time.Time
	Reason    string
}

func NewStageTransition(
	fromStage PipelineStage,
	toStage PipelineStage,
	changedBy uuid.UUID,
	reason string,
	now time.Time,
) (StageTransition, error) {
	if changedBy == uuid.Nil {
		return StageTransition{}, ErrInvalidTransitionActor
	}
	if !toStage.Valid() {
		return StageTransition{}, ErrInvalidTransitionStage
	}

	return StageTransition{
		ID:        uuid.New(),
		FromStage: fromStage,
		ToStage:   toStage,
		ChangedBy: changedBy,
		ChangedAt: now,
		Reason:    strings.TrimSpace(reason),
	}, nil
}

func (t StageTransition) IsInitial() bool {
	return t.FromStage == "" && t.ToStage == StageApplied
}

func (t StageTransition) IsTerminal() bool {
	return t.ToStage.IsTerminal()
}

type RejectCode string

const (
	RejectCodeNotQualified      RejectCode = "NOT_QUALIFIED"
	RejectCodeAssessmentFailed  RejectCode = "ASSESSMENT_FAILED"
	RejectCodeCultureMismatch   RejectCode = "CULTURE_MISMATCH"
	RejectCodeCandidateWithdrew RejectCode = "CANDIDATE_WITHDREW"
	RejectCodeVacancyClosed     RejectCode = "VACANCY_CLOSED"
	RejectCodeOverqualified     RejectCode = "OVERQUALIFIED"
	RejectCodeOther             RejectCode = "OTHER"
)

func (c RejectCode) String() string {
	return string(c)
}

func (c RejectCode) Valid() bool {
	switch c {
	case RejectCodeNotQualified,
		RejectCodeAssessmentFailed,
		RejectCodeCultureMismatch,
		RejectCodeCandidateWithdrew,
		RejectCodeVacancyClosed,
		RejectCodeOverqualified,
		RejectCodeOther:
		return true
	default:
		return false
	}
}

type RejectReason struct {
	Code    RejectCode
	Comment string
}

func NewRejectReason(code RejectCode, comment string) (RejectReason, error) {
	comment = strings.TrimSpace(comment)

	if !code.Valid() && comment == "" {
		return RejectReason{}, ErrEmptyRejectReason
	}
	if !code.Valid() {
		code = RejectCodeOther
	}

	return RejectReason{
		Code:    code,
		Comment: comment,
	}, nil
}

func (r RejectReason) EffectiveComment() string {
	if r.Comment != "" {
		return r.Comment
	}
	return string(r.Code)
}
