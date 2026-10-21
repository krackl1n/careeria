package interview

import (
	"time"

	"github.com/google/uuid"
)

type InterviewStatus string

const (
	StatusScheduled InterviewStatus = "SCHEDULED"
	StatusCompleted InterviewStatus = "COMPLETED"
	StatusCancelled InterviewStatus = "CANCELLED"
)

func (s InterviewStatus) String() string {
	return string(s)
}

func (s InterviewStatus) Valid() bool {
	switch s {
	case StatusScheduled, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

type MeetingFormat string

const (
	FormatOnlineLiveKit MeetingFormat = "ONLINE_LIVEKIT"
	FormatOnsite        MeetingFormat = "ONSITE"
	FormatExternalLink  MeetingFormat = "EXTERNAL_LINK"
)

func (f MeetingFormat) String() string {
	return string(f)
}

func (f MeetingFormat) Valid() bool {
	switch f {
	case FormatOnlineLiveKit, FormatOnsite, FormatExternalLink:
		return true
	default:
		return false
	}
}

type Participant struct {
	UserID uuid.UUID
	Role   string
}

type Feedback struct {
	InterviewerUserID uuid.UUID
	Score             int
	Notes             string
	SubmittedAt       time.Time
}

func (f Feedback) Validate() error {
	if f.InterviewerUserID == uuid.Nil {
		return ErrEmptyInterviewerID
	}
	if f.Score < 1 || f.Score > 5 {
		return ErrInvalidScore
	}
	return nil
}

type InterviewSummary struct {
	InterviewID   uuid.UUID
	ApplicationID uuid.UUID
	Stage         string
	Status        InterviewStatus
	HasFeedback   bool
	Score         int
	CompletedAt   time.Time
}

