package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	policydomain "github.com/krackl1n/careeria/services/hiring/internal/domain/policy"
	"github.com/stretchr/testify/require"
)

type mockAppRepo struct {
	apps map[uuid.UUID]*applicationdomain.JobApplication
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{apps: make(map[uuid.UUID]*applicationdomain.JobApplication)}
}

func (m *mockAppRepo) Create(_ context.Context, app *applicationdomain.JobApplication) error {
	m.apps[app.ID()] = app
	return nil
}

func (m *mockAppRepo) Update(_ context.Context, app *applicationdomain.JobApplication) error {
	m.apps[app.ID()] = app
	return nil
}

func (m *mockAppRepo) GetByID(_ context.Context, id uuid.UUID) (*applicationdomain.JobApplication, error) {
	app, ok := m.apps[id]
	if !ok {
		return nil, applicationdomain.ErrNotFound
	}
	return app, nil
}

func (m *mockAppRepo) GetActiveByCandidateAndVacancy(_ context.Context, candidateID, vacancyID uuid.UUID) (*applicationdomain.JobApplication, error) {
	for _, app := range m.apps {
		if app.CandidateID() == candidateID && app.VacancyID() == vacancyID && !app.CurrentStage().IsTerminal() {
			return app, nil
		}
	}
	return nil, applicationdomain.ErrNotFound
}

type mockInterviewSummaryRepo struct {
	summaries map[uuid.UUID][]interviewdomain.InterviewSummary
}

func (m *mockInterviewSummaryRepo) FindByApplicationID(_ context.Context, appID uuid.UUID) ([]interviewdomain.InterviewSummary, error) {
	return m.summaries[appID], nil
}

type mockOutboxRepo struct {
	messages []eventdomain.Message
}

func (m *mockOutboxRepo) Save(_ context.Context, msg eventdomain.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

type mockAuthPort struct {
	deny bool
}

func (m *mockAuthPort) CanApplyVacancy(_ context.Context, _, _ uuid.UUID) error {
	if m.deny {
		return apperr.Forbidden("forbidden", "cannot apply", nil)
	}
	return nil
}

func (m *mockAuthPort) CanManageApplication(_ context.Context, _, _ uuid.UUID) error {
	if m.deny {
		return apperr.Forbidden("forbidden", "cannot manage", nil)
	}
	return nil
}

type mockTxManager struct{}

func (m *mockTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequentialIDGen struct {
	ids []uuid.UUID
	idx int
}

func (g *sequentialIDGen) New() uuid.UUID {
	if g.idx < len(g.ids) {
		id := g.ids[g.idx]
		g.idx++
		return id
	}
	return uuid.New()
}

func TestApplicationService(t *testing.T) {
	ctx := context.Background()
	appRepo := newMockAppRepo()
	intSumRepo := &mockInterviewSummaryRepo{summaries: make(map[uuid.UUID][]interviewdomain.InterviewSummary)}
	outboxRepo := &mockOutboxRepo{}
	authPort := &mockAuthPort{}
	policy := policydomain.NewOfferEligibilityPolicy()
	txMgr := &mockTxManager{}
	now := time.Now().UTC()
	clock := fixedClock{now: now}
	idGen := &sequentialIDGen{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}}
	eventFactory := eventdomain.NewFactory(idGen)

	svc := NewService(appRepo, intSumRepo, outboxRepo, authPort, policy, txMgr, eventFactory, clock, idGen)

	candidateID := uuid.New()
	vacancyID := uuid.New()

	appID, err := svc.ApplyVacancy(ctx, ApplyVacancyInput{
		VacancyID:   vacancyID,
		CandidateID: candidateID,
		ActorID:     candidateID,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, appID)
	require.Len(t, outboxRepo.messages, 1)

	_, err = svc.ApplyVacancy(ctx, ApplyVacancyInput{
		VacancyID:   vacancyID,
		CandidateID: candidateID,
		ActorID:     candidateID,
	})
	require.Error(t, err)
	require.True(t, apperr.IsType(err, apperr.TypeConflict))

	operatorID := uuid.New()
	err = svc.ChangeStage(ctx, ChangeStageInput{
		ApplicationID: appID,
		TargetStage:   applicationdomain.StageScreening,
		ActorID:       operatorID,
		Reason:        "passed initial filter",
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 2)

	err = svc.PromoteToInterview(ctx, appID, operatorID, "screening approved")
	require.NoError(t, err)

	err = svc.InitiateOffer(ctx, appID, operatorID, "all interviews completed")
	require.Error(t, err)

	intSumRepo.summaries[appID] = []interviewdomain.InterviewSummary{
		{
			InterviewID: uuid.New(),
			Status:      interviewdomain.StatusCompleted,
			HasFeedback: true,
			Score:       5,
		},
	}

	err = svc.InitiateOffer(ctx, appID, operatorID, "all interviews completed")
	require.NoError(t, err)

	history, err := svc.GetPipelineHistory(ctx, appID, operatorID)
	require.NoError(t, err)
	require.Equal(t, applicationdomain.StageOfferPending, history.CurrentStage)
	require.Len(t, history.History, 4)
}

