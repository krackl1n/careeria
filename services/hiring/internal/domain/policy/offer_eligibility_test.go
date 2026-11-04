package policy

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/stretchr/testify/require"
)

func TestOfferEligibilityPolicy(t *testing.T) {
	policy := NewOfferEligibilityPolicy()
	now := time.Now().UTC()
	actorID := uuid.New()

	app, err := application.New(uuid.New(), uuid.New(), uuid.New(), actorID, now)
	require.NoError(t, err)

	// In APPLIED stage - should fail
	err = policy.CanInitiateOffer(app, nil)
	require.ErrorIs(t, err, ErrInvalidStageForOffer)

	// Move to SCREENING -> INTERVIEW
	err = app.ChangeStage(application.StageScreening, actorID, "screening", now)
	require.NoError(t, err)
	err = app.MarkInterview(actorID, "invited to interview", now)
	require.NoError(t, err)

	// In INTERVIEW stage, but no interviews - should fail
	err = policy.CanInitiateOffer(app, nil)
	require.ErrorIs(t, err, ErrMissingCompletedInterview)

	// In INTERVIEW stage, with only SCHEDULED interview - should fail
	itScheduled, err := interview.Schedule(
		uuid.New(), uuid.New(), interview.TypeTechnical, interview.FormatOnlineLiveKit,
		now, now.Add(time.Hour),
		[]interview.Participant{{UserID: uuid.New(), Role: interview.RoleLeadInterviewer}},
		"https://meet.careeria.com/room",
		now,
	)
	require.NoError(t, err)
	err = policy.CanInitiateOffer(app, []*interview.Interview{itScheduled})
	require.ErrorIs(t, err, ErrMissingCompletedInterview)

	// In INTERVIEW stage, with COMPLETED interview, but no feedback - should fail
	itCompletedWithoutFeedback, err := interview.Schedule(
		uuid.New(), uuid.New(), interview.TypeTechnical, interview.FormatOnlineLiveKit,
		now, now.Add(time.Hour),
		[]interview.Participant{{UserID: uuid.New(), Role: interview.RoleLeadInterviewer}},
		"https://meet.careeria.com/room",
		now,
	)
	require.NoError(t, err)
	err = itCompletedWithoutFeedback.Complete(now)
	require.NoError(t, err)
	err = policy.CanInitiateOffer(app, []*interview.Interview{itCompletedWithoutFeedback})
	require.ErrorIs(t, err, ErrMissingCompletedInterview)

	// In INTERVIEW stage, with COMPLETED interview and feedback - should PASS
	interviewerID := uuid.New()
	itWithFeedback, err := interview.Schedule(
		uuid.New(), uuid.New(), interview.TypeTechnical, interview.FormatOnlineLiveKit,
		now, now.Add(time.Hour),
		[]interview.Participant{{UserID: interviewerID, Role: interview.RoleLeadInterviewer}},
		"https://meet.careeria.com/room",
		now,
	)
	require.NoError(t, err)
	fb, err := interview.NewFeedback(interviewerID, 4, "Strong candidate", now)
	require.NoError(t, err)
	err = itWithFeedback.SubmitFeedback(fb, now)
	require.NoError(t, err)
	err = itWithFeedback.Complete(now)
	require.NoError(t, err)
	err = policy.CanInitiateOffer(app, []*interview.Interview{itWithFeedback})
	require.NoError(t, err)
}
