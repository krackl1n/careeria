package interview

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Interview struct {
	id                 uuid.UUID
	applicationID      uuid.UUID
	interviewType      string
	status             InterviewStatus
	scheduledStartTime time.Time
	scheduledEndTime   time.Time
	meetingRoomURL     string
	participants       []Participant
	feedbacks          []Feedback
	version            int64
	createdAt          time.Time
	updatedAt          time.Time
}

func Schedule(
	id uuid.UUID,
	applicationID uuid.UUID,
	interviewType string,
	scheduledStartTime time.Time,
	scheduledEndTime time.Time,
	participants []Participant,
	meetingRoomURL string,
	now time.Time,
) (*Interview, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if applicationID == uuid.Nil {
		return nil, ErrEmptyApplicationID
	}
	if strings.TrimSpace(interviewType) == "" {
		return nil, ErrEmptyInterviewType
	}
	if scheduledStartTime.Before(now) {
		return nil, ErrStartInPast
	}
	if !scheduledEndTime.After(scheduledStartTime) {
		return nil, ErrInvalidInterval
	}
	if len(participants) == 0 {
		return nil, ErrEmptyParticipants
	}

	for _, p := range participants {
		if p.UserID == uuid.Nil {
			return nil, ErrInvalidParticipant
		}
	}

	copiedParticipants := make([]Participant, len(participants))
	copy(copiedParticipants, participants)

	return &Interview{
		id:                 id,
		applicationID:      applicationID,
		interviewType:      strings.TrimSpace(interviewType),
		status:             StatusScheduled,
		scheduledStartTime: scheduledStartTime,
		scheduledEndTime:   scheduledEndTime,
		meetingRoomURL:     strings.TrimSpace(meetingRoomURL),
		participants:       copiedParticipants,
		feedbacks:          make([]Feedback, 0),
		version:            1,
		createdAt:          now,
		updatedAt:          now,
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	applicationID uuid.UUID,
	interviewType string,
	status InterviewStatus,
	scheduledStartTime time.Time,
	scheduledEndTime time.Time,
	meetingRoomURL string,
	participants []Participant,
	feedbacks []Feedback,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) *Interview {
	return &Interview{
		id:                 id,
		applicationID:      applicationID,
		interviewType:      interviewType,
		status:             status,
		scheduledStartTime: scheduledStartTime,
		scheduledEndTime:   scheduledEndTime,
		meetingRoomURL:     meetingRoomURL,
		participants:       participants,
		feedbacks:          feedbacks,
		version:            version,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
	}
}

func (i *Interview) ID() uuid.UUID {
	return i.id
}

func (i *Interview) ApplicationID() uuid.UUID {
	return i.applicationID
}

func (i *Interview) InterviewType() string {
	return i.interviewType
}

func (i *Interview) Status() InterviewStatus {
	return i.status
}

func (i *Interview) ScheduledStartTime() time.Time {
	return i.scheduledStartTime
}

func (i *Interview) ScheduledEndTime() time.Time {
	return i.scheduledEndTime
}

func (i *Interview) MeetingRoomURL() string {
	return i.meetingRoomURL
}

func (i *Interview) Participants() []Participant {
	copied := make([]Participant, len(i.participants))
	copy(copied, i.participants)
	return copied
}

func (i *Interview) Feedbacks() []Feedback {
	copied := make([]Feedback, len(i.feedbacks))
	copy(copied, i.feedbacks)
	return copied
}

func (i *Interview) Version() int64 {
	return i.version
}

func (i *Interview) CreatedAt() time.Time {
	return i.createdAt
}

func (i *Interview) UpdatedAt() time.Time {
	return i.updatedAt
}

func (i *Interview) SetMeetingRoomURL(url string, now time.Time) {
	i.meetingRoomURL = strings.TrimSpace(url)
	i.version++
	i.updatedAt = now
}

func (i *Interview) Complete(now time.Time) error {
	if i.status != StatusScheduled {
		return fmt.Errorf("%w: cannot complete interview in status %s", ErrInvalidStatus, i.status)
	}

	i.status = StatusCompleted
	i.version++
	i.updatedAt = now
	return nil
}

func (i *Interview) Cancel(reason string, now time.Time) error {
	if i.status != StatusScheduled {
		return fmt.Errorf("%w: cannot cancel interview in status %s", ErrInvalidStatus, i.status)
	}

	i.status = StatusCancelled
	i.version++
	i.updatedAt = now
	return nil
}

func (i *Interview) SubmitFeedback(feedback Feedback, now time.Time) error {
	if err := feedback.Validate(); err != nil {
		return err
	}

	if i.status == StatusCancelled {
		return ErrInterviewCancelled
	}

	isParticipant := false
	for _, p := range i.participants {
		if p.UserID == feedback.InterviewerUserID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	for _, f := range i.feedbacks {
		if f.InterviewerUserID == feedback.InterviewerUserID {
			return ErrFeedbackAlreadySubmitted
		}
	}

	feedback.SubmittedAt = now
	i.feedbacks = append(i.feedbacks, feedback)
	i.version++
	i.updatedAt = now
	return nil
}

func (i *Interview) ToSummary() InterviewSummary {
	hasFeedback := len(i.feedbacks) > 0
	avgScore := 0
	if hasFeedback {
		total := 0
		for _, f := range i.feedbacks {
			total += f.Score
		}
		avgScore = total / len(i.feedbacks)
	}

	return InterviewSummary{
		InterviewID:   i.id,
		ApplicationID: i.applicationID,
		Stage:         i.interviewType,
		Status:        i.status,
		HasFeedback:   hasFeedback,
		Score:         avgScore,
		CompletedAt:   i.updatedAt,
	}
}
