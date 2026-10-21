package interview

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Interview struct {
	ID                 uuid.UUID
	ApplicationID      uuid.UUID
	InterviewType      string
	Format             MeetingFormat
	Status             InterviewStatus
	ScheduledStartTime time.Time
	ScheduledEndTime   time.Time
	MeetingRoomURL     string
	Participants       []Participant
	Feedbacks          []Feedback
	Version            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func Schedule(
	id uuid.UUID,
	applicationID uuid.UUID,
	interviewType string,
	format MeetingFormat,
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
	if !format.Valid() {
		return nil, ErrInvalidFormat
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
		ID:                 id,
		ApplicationID:      applicationID,
		InterviewType:      strings.TrimSpace(interviewType),
		Format:             format,
		Status:             StatusScheduled,
		ScheduledStartTime: scheduledStartTime,
		ScheduledEndTime:   scheduledEndTime,
		MeetingRoomURL:     strings.TrimSpace(meetingRoomURL),
		Participants:       copiedParticipants,
		Feedbacks:          make([]Feedback, 0),
		Version:            1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	applicationID uuid.UUID,
	interviewType string,
	format MeetingFormat,
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
		ID:                 id,
		ApplicationID:      applicationID,
		InterviewType:      interviewType,
		Format:             format,
		Status:             status,
		ScheduledStartTime: scheduledStartTime,
		ScheduledEndTime:   scheduledEndTime,
		MeetingRoomURL:     meetingRoomURL,
		Participants:       participants,
		Feedbacks:          feedbacks,
		Version:            version,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}
}

func (i *Interview) SetMeetingRoomURL(url string, now time.Time) {
	i.MeetingRoomURL = strings.TrimSpace(url)
	i.Version++
	i.UpdatedAt = now
}

func (i *Interview) Complete(now time.Time) error {
	if i.Status != StatusScheduled {
		return fmt.Errorf("%w: cannot complete interview in status %s", ErrInvalidStatus, i.Status)
	}

	i.Status = StatusCompleted
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i *Interview) Cancel(reason string, now time.Time) error {
	if i.Status != StatusScheduled {
		return fmt.Errorf("%w: cannot cancel interview in status %s", ErrInvalidStatus, i.Status)
	}

	i.Status = StatusCancelled
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i *Interview) SubmitFeedback(feedback Feedback, now time.Time) error {
	if err := feedback.Validate(); err != nil {
		return err
	}

	if i.Status == StatusCancelled {
		return ErrInterviewCancelled
	}

	isParticipant := false
	for _, p := range i.Participants {
		if p.UserID == feedback.InterviewerUserID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	for _, f := range i.Feedbacks {
		if f.InterviewerUserID == feedback.InterviewerUserID {
			return ErrFeedbackAlreadySubmitted
		}
	}

	feedback.SubmittedAt = now
	i.Feedbacks = append(i.Feedbacks, feedback)
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i *Interview) ToSummary() InterviewSummary {
	hasFeedback := len(i.Feedbacks) > 0
	avgScore := 0
	if hasFeedback {
		total := 0
		for _, f := range i.Feedbacks {
			total += f.Score
		}
		avgScore = total / len(i.Feedbacks)
	}

	return InterviewSummary{
		InterviewID:   i.ID,
		ApplicationID: i.ApplicationID,
		Stage:         i.InterviewType,
		Status:        i.Status,
		HasFeedback:   hasFeedback,
		Score:         avgScore,
		CompletedAt:   i.UpdatedAt,
	}
}
