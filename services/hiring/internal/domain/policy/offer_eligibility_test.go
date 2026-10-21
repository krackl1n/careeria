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
	scheduled := []interview.InterviewSummary{
		{
			InterviewID: uuid.New(),
			Status:      interview.StatusScheduled,
			HasFeedback: false,
		},
	}
	err = policy.CanInitiateOffer(app, scheduled)
	require.ErrorIs(t, err, ErrMissingCompletedInterview)

	// In INTERVIEW stage, with COMPLETED interview, but no feedback - should fail
	completedWithoutFeedback := []interview.InterviewSummary{
		{
			InterviewID: uuid.New(),
			Status:      interview.StatusCompleted,
			HasFeedback: false,
		},
	}
	err = policy.CanInitiateOffer(app, completedWithoutFeedback)
	require.ErrorIs(t, err, ErrMissingCompletedInterview)

	// In INTERVIEW stage, with COMPLETED interview and feedback - should PASS
	completedWithFeedback := []interview.InterviewSummary{
		{
			InterviewID: uuid.New(),
			Status:      interview.StatusCompleted,
			HasFeedback: true,
			Score:       4,
		},
	}
	err = policy.CanInitiateOffer(app, completedWithFeedback)
	require.NoError(t, err)
}

