package interview

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/event"
)

const (
	AggregateTypeInterview event.AggregateType = "interview"

	EventTypeInterviewScheduled event.Type = "hiring.interview.scheduled"
	EventTypeInterviewCompleted event.Type = "hiring.interview.completed"
	EventTypeInterviewCancelled event.Type = "hiring.interview.cancelled"
	EventTypeFeedbackSubmitted  event.Type = "hiring.interview.feedback_submitted"
)

type InterviewScheduledPayload struct {
	InterviewID   uuid.UUID     `json:"interview_id"`
	ApplicationID uuid.UUID     `json:"application_id"`
	InterviewType string        `json:"interview_type"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Participants  []Participant `json:"participants"`
}

type InterviewCompletedPayload struct {
	InterviewID uuid.UUID `json:"interview_id"`
	CompletedAt time.Time `json:"completed_at"`
}

type InterviewCancelledPayload struct {
	InterviewID uuid.UUID `json:"interview_id"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

type FeedbackSubmittedPayload struct {
	InterviewID       uuid.UUID `json:"interview_id"`
	InterviewerUserID uuid.UUID `json:"interviewer_user_id"`
	Score             int       `json:"score"`
	SubmittedAt       time.Time `json:"submitted_at"`
}

