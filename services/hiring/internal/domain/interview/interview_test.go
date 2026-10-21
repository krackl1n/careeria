package interview

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInterviewLifecycle(t *testing.T) {
	now := time.Now().UTC()
	interviewID := uuid.New()
	applicationID := uuid.New()
	interviewer1 := uuid.New()
	interviewer2 := uuid.New()

	startTime := now.Add(2 * time.Hour)
	endTime := startTime.Add(time.Hour)

	participants := []Participant{
		{UserID: interviewer1, Role: "tech_lead"},
		{UserID: interviewer2, Role: "senior_dev"},
	}

	// Schedule interview
	it, err := Schedule(
		interviewID,
		applicationID,
		"TECHNICAL",
		FormatOnlineLiveKit,
		startTime,
		endTime,
		participants,
		"https://livekit.example/room/123",
		now,
	)
	require.NoError(t, err)
	require.Equal(t, FormatOnlineLiveKit, it.Format)
	require.Equal(t, StatusScheduled, it.Status)
	require.Equal(t, int64(1), it.Version)

	// Validate cannot schedule in the past
	_, err = Schedule(
		uuid.New(),
		applicationID,
		"HR",
		FormatOnlineLiveKit,
		now.Add(-time.Hour),
		now,
		participants,
		"",
		now,
	)
	require.ErrorIs(t, err, ErrStartInPast)

	// Validate cannot submit feedback from non-participant
	err = it.SubmitFeedback(Feedback{
		InterviewerUserID: uuid.New(),
		Score:             5,
		Notes:             "Super candidate",
	}, now)
	require.ErrorIs(t, err, ErrNotParticipant)

	// Submit valid feedback
	err = it.SubmitFeedback(Feedback{
		InterviewerUserID: interviewer1,
		Score:             4,
		Notes:             "Good knowledge of Go and system design",
	}, now)
	require.NoError(t, err)
	require.Len(t, it.Feedbacks, 1)
	require.Equal(t, int64(2), it.Version)

	// Forbid duplicate feedback
	err = it.SubmitFeedback(Feedback{
		InterviewerUserID: interviewer1,
		Score:             5,
		Notes:             "Changed mind",
	}, now)
	require.ErrorIs(t, err, ErrFeedbackAlreadySubmitted)

	// Complete interview
	err = it.Complete(now.Add(3 * time.Hour))
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, it.Status)

	// Check summary projection
	summary := it.ToSummary()
	require.Equal(t, interviewID, summary.InterviewID)
	require.Equal(t, StatusCompleted, summary.Status)
	require.True(t, summary.HasFeedback)
	require.Equal(t, 4, summary.Score)
}

func TestInterviewCancellation(t *testing.T) {
	now := time.Now().UTC()
	startTime := now.Add(2 * time.Hour)
	endTime := startTime.Add(time.Hour)

	it, err := Schedule(
		uuid.New(),
		uuid.New(),
		"BEHAVIORAL",
		FormatOnsite,
		startTime,
		endTime,
		[]Participant{{UserID: uuid.New(), Role: "recruiter"}},
		"Office Moscow, Room 101",
		now,
	)
	require.NoError(t, err)

	err = it.Cancel("candidate requested reschedule", now)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, it.Status)

	err = it.SubmitFeedback(Feedback{
		InterviewerUserID: it.Participants[0].UserID,
		Score:             3,
	}, now)
	require.ErrorIs(t, err, ErrInterviewCancelled)
}
