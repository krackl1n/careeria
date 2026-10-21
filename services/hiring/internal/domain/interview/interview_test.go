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
	observer := uuid.New()

	startTime := now.Add(2 * time.Hour)
	endTime := startTime.Add(time.Hour)

	p1, err := NewParticipant(interviewer1, RoleLeadInterviewer)
	require.NoError(t, err)
	p2, err := NewParticipant(interviewer2, RoleInterviewer)
	require.NoError(t, err)
	pObs, err := NewParticipant(observer, RoleObserver)
	require.NoError(t, err)

	participants := []Participant{p1, p2, pObs}

	it, err := Schedule(
		interviewID,
		applicationID,
		TypeTechnical,
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

	_, err = Schedule(
		uuid.New(),
		applicationID,
		TypeScreening,
		FormatOnlineLiveKit,
		now.Add(-time.Hour),
		now,
		participants,
		"",
		now,
	)
	require.ErrorIs(t, err, ErrStartInPast)

	fbNonParticipant, err := NewFeedback(uuid.New(), 5, "Super candidate", now)
	require.NoError(t, err)
	err = it.SubmitFeedback(fbNonParticipant, now)
	require.ErrorIs(t, err, ErrNotParticipant)

	fbObserver, err := NewFeedback(observer, 4, "Notes from observer", now)
	require.NoError(t, err)
	err = it.SubmitFeedback(fbObserver, now)
	require.ErrorIs(t, err, ErrParticipantCannotEvaluate)

	fbValid, err := NewFeedback(interviewer1, 4, "Good knowledge of Go and system design", now)
	require.NoError(t, err)
	err = it.SubmitFeedback(fbValid, now)
	require.NoError(t, err)
	require.Len(t, it.Feedbacks, 1)
	require.Equal(t, int64(2), it.Version)

	err = it.SubmitFeedback(fbValid, now)
	require.ErrorIs(t, err, ErrFeedbackAlreadySubmitted)

	err = it.Complete(now.Add(3 * time.Hour))
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, it.Status)

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

	recruiter, err := NewParticipant(uuid.New(), RoleRecruiter)
	require.NoError(t, err)

	it, err := Schedule(
		uuid.New(),
		uuid.New(),
		TypeBehavioral,
		FormatOnsite,
		startTime,
		endTime,
		[]Participant{recruiter},
		"Office Moscow, Room 101",
		now,
	)
	require.NoError(t, err)

	err = it.Cancel("candidate requested reschedule", now)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, it.Status)

	fb, err := NewFeedback(recruiter.UserID, 3, "", now)
	require.NoError(t, err)
	err = it.SubmitFeedback(fb, now)
	require.ErrorIs(t, err, ErrInterviewCancelled)
}
