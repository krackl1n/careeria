package application

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJobApplicationLifecycle(t *testing.T) {
	appID := uuid.New()
	vacancyID := uuid.New()
	candidateID := uuid.New()
	actorID := uuid.New()
	now := time.Now().UTC()

	app, err := New(appID, vacancyID, candidateID, actorID, now)
	require.NoError(t, err)
	require.Equal(t, StageApplied, app.CurrentStage)
	require.Equal(t, int64(1), app.Version)
	require.Len(t, app.History, 1)

	// Invalid transition directly from APPLIED to HIRED
	err = app.ChangeStage(StageHired, actorID, "direct hire", now)
	require.ErrorIs(t, err, ErrInvalidStageTransition)

	// APPLIED -> SCREENING
	err = app.ChangeStage(StageScreening, actorID, "screening started", now)
	require.NoError(t, err)
	require.Equal(t, StageScreening, app.CurrentStage)
	require.Equal(t, int64(2), app.Version)

	// SCREENING -> TECH_ASSESSMENT
	err = app.ChangeStage(StageTechAssessment, actorID, "test task assigned", now)
	require.NoError(t, err)
	require.Equal(t, StageTechAssessment, app.CurrentStage)

	// TECH_ASSESSMENT -> INTERVIEW (via AdvanceFromAssessment)
	err = app.AdvanceFromAssessment(90, actorID, now)
	require.NoError(t, err)
	require.Equal(t, StageInterview, app.CurrentStage)

	// INTERVIEW -> OFFER_PENDING
	err = app.MarkOfferPending(actorID, now)
	require.NoError(t, err)
	require.Equal(t, StageOfferPending, app.CurrentStage)

	// OFFER_PENDING -> OFFER_ACCEPTED
	err = app.ChangeStage(StageOfferAccepted, actorID, "candidate accepted offer", now)
	require.NoError(t, err)
	require.Equal(t, StageOfferAccepted, app.CurrentStage)

	// OFFER_ACCEPTED -> HIRED
	err = app.ChangeStage(StageHired, actorID, "onboarded", now)
	require.NoError(t, err)
	require.Equal(t, StageHired, app.CurrentStage)
	require.True(t, app.CurrentStage.IsTerminal())

	// From terminal HIRED, cannot transition to any other stage
	err = app.ChangeStage(StageInterview, actorID, "try again", now)
	require.ErrorIs(t, err, ErrInvalidStageTransition)
	err = app.Reject(RejectReason{Code: "reason"}, actorID, now)
	require.ErrorIs(t, err, ErrTerminalStage)
}

func TestJobApplicationRejectionAndWithdrawal(t *testing.T) {
	appID := uuid.New()
	vacancyID := uuid.New()
	candidateID := uuid.New()
	actorID := uuid.New()
	now := time.Now().UTC()

	t.Run("reject during screening", func(t *testing.T) {
		app, err := New(appID, vacancyID, candidateID, actorID, now)
		require.NoError(t, err)

		err = app.ChangeStage(StageScreening, actorID, "screening", now)
		require.NoError(t, err)

		err = app.Reject(RejectReason{Code: "not_qualified", Comment: "Insufficient experience"}, actorID, now)
		require.NoError(t, err)
		require.Equal(t, StageRejected, app.CurrentStage)
		require.True(t, app.CurrentStage.IsTerminal())

		// Cannot transition after rejection
		err = app.ChangeStage(StageInterview, actorID, "oops", now)
		require.ErrorIs(t, err, ErrInvalidStageTransition)
	})

	t.Run("withdraw during interview", func(t *testing.T) {
		app, err := New(appID, vacancyID, candidateID, actorID, now)
		require.NoError(t, err)

		err = app.ChangeStage(StageScreening, actorID, "screening", now)
		require.NoError(t, err)

		err = app.MarkInterview(actorID, "interview scheduled", now)
		require.NoError(t, err)

		err = app.Withdraw(actorID, "accepted another offer", now)
		require.NoError(t, err)
		require.Equal(t, StageWithdrawn, app.CurrentStage)
		require.True(t, app.CurrentStage.IsTerminal())
	})
}
