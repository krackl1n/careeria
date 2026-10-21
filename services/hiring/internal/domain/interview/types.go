package interview

import (
	"fmt"
	"strings"
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

type InterviewType string

const (
	TypeScreening    InterviewType = "SCREENING"
	TypeTechnical    InterviewType = "TECHNICAL"
	TypeSystemDesign InterviewType = "SYSTEM_DESIGN"
	TypeBehavioral   InterviewType = "BEHAVIORAL"
	TypeBarRaiser    InterviewType = "BAR_RAISER"
	TypeExecutive    InterviewType = "EXECUTIVE"
)

func (t InterviewType) String() string {
	return string(t)
}

func (t InterviewType) Valid() bool {
	switch t {
	case TypeScreening, TypeTechnical, TypeSystemDesign, TypeBehavioral, TypeBarRaiser, TypeExecutive:
		return true
	default:
		return false
	}
}

func ParseInterviewType(value string) (InterviewType, error) {
	it := InterviewType(strings.ToUpper(strings.TrimSpace(value)))
	if !it.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidInterviewType, value)
	}
	return it, nil
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

type ParticipantRole string

const (
	RoleLeadInterviewer ParticipantRole = "LEAD_INTERVIEWER"
	RoleInterviewer     ParticipantRole = "INTERVIEWER"
	RoleHiringManager   ParticipantRole = "HIRING_MANAGER"
	RoleRecruiter       ParticipantRole = "RECRUITER"
	RoleBarRaiser       ParticipantRole = "BAR_RAISER"
	RoleObserver        ParticipantRole = "OBSERVER"
	RoleCandidate       ParticipantRole = "CANDIDATE"
)

func (r ParticipantRole) String() string {
	return string(r)
}

func (r ParticipantRole) Valid() bool {
	switch r {
	case RoleLeadInterviewer,
		RoleInterviewer,
		RoleHiringManager,
		RoleRecruiter,
		RoleBarRaiser,
		RoleObserver,
		RoleCandidate:
		return true
	default:
		return false
	}
}

func (r ParticipantRole) CanEvaluate() bool {
	switch r {
	case RoleLeadInterviewer, RoleInterviewer, RoleHiringManager, RoleRecruiter, RoleBarRaiser:
		return true
	default:
		return false
	}
}

func ParseParticipantRole(value string) (ParticipantRole, error) {
	role := ParticipantRole(strings.ToUpper(strings.TrimSpace(value)))
	if !role.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidParticipantRole, value)
	}
	return role, nil
}

type Participant struct {
	UserID uuid.UUID
	Role   ParticipantRole
}

func NewParticipant(userID uuid.UUID, role ParticipantRole) (Participant, error) {
	if userID == uuid.Nil {
		return Participant{}, ErrInvalidParticipant
	}
	if !role.Valid() {
		return Participant{}, ErrInvalidParticipantRole
	}
	return Participant{
		UserID: userID,
		Role:   role,
	}, nil
}

func (p Participant) IsInterviewer() bool {
	return p.Role.CanEvaluate()
}

type Feedback struct {
	InterviewerUserID uuid.UUID
	Score             int
	Notes             string
	SubmittedAt       time.Time
}

func NewFeedback(interviewerUserID uuid.UUID, score int, notes string, now time.Time) (Feedback, error) {
	if interviewerUserID == uuid.Nil {
		return Feedback{}, ErrEmptyInterviewerID
	}
	if score < 1 || score > 5 {
		return Feedback{}, ErrInvalidScore
	}
	return Feedback{
		InterviewerUserID: interviewerUserID,
		Score:             score,
		Notes:             strings.TrimSpace(notes),
		SubmittedAt:       now,
	}, nil
}

func (f Feedback) IsPassing() bool {
	return f.Score >= 3
}

func (f Feedback) IsStrongPass() bool {
	return f.Score >= 4
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
	Stage         InterviewType
	Status        InterviewStatus
	HasFeedback   bool
	Score         int
	CompletedAt   time.Time
}
