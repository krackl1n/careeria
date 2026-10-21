package interview

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/stretchr/testify/require"
)

type mockInterviewRepo struct {
	interviews map[uuid.UUID]*interviewdomain.Interview
}

func newMockInterviewRepo() *mockInterviewRepo {
	return &mockInterviewRepo{interviews: make(map[uuid.UUID]*interviewdomain.Interview)}
}

func (m *mockInterviewRepo) Create(_ context.Context, it *interviewdomain.Interview) error {
	m.interviews[it.ID()] = it
	return nil
}

func (m *mockInterviewRepo) Update(_ context.Context, it *interviewdomain.Interview) error {
	m.interviews[it.ID()] = it
	return nil
}

func (m *mockInterviewRepo) GetByID(_ context.Context, id uuid.UUID) (*interviewdomain.Interview, error) {
	it, ok := m.interviews[id]
	if !ok {
		return nil, interviewdomain.ErrNotFound
	}
	return it, nil
}

func (m *mockInterviewRepo) ListByApplicationID(_ context.Context, appID uuid.UUID) ([]*interviewdomain.Interview, error) {
	var list []*interviewdomain.Interview
	for _, it := range m.interviews {
		if it.ApplicationID() == appID {
			list = append(list, it)
		}
	}
	return list, nil
}

type mockAppStageChecker struct {
	stage applicationdomain.PipelineStage
}

func (m *mockAppStageChecker) GetStage(_ context.Context, _ uuid.UUID) (applicationdomain.PipelineStage, error) {
	return m.stage, nil
}

type mockConferenceSvc struct {
	createdRooms []string
	endedRooms   []string
}

func (m *mockConferenceSvc) CreateRoom(_ context.Context, roomName string) (string, error) {
	m.createdRooms = append(m.createdRooms, roomName)
	return "https://livekit.example/rooms/" + roomName, nil
}

func (m *mockConferenceSvc) EndRoom(_ context.Context, roomName string) error {
	m.endedRooms = append(m.endedRooms, roomName)
	return nil
}

type mockOutboxRepo struct {
	messages []eventdomain.Message
}

func (m *mockOutboxRepo) Save(_ context.Context, msg eventdomain.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

type mockAuthPort struct{}

func (m *mockAuthPort) CanManageInterview(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockAuthPort) CanSubmitFeedback(_ context.Context, _, _ uuid.UUID) error {
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

type fixedIDGen struct {
	id uuid.UUID
}

func (g fixedIDGen) New() uuid.UUID {
	return g.id
}

func TestInterviewService(t *testing.T) {
	ctx := context.Background()
	itRepo := newMockInterviewRepo()
	appChecker := &mockAppStageChecker{stage: applicationdomain.StageInterview}
	confSvc := &mockConferenceSvc{}
	outboxRepo := &mockOutboxRepo{}
	authPort := &mockAuthPort{}
	txMgr := &mockTxManager{}

	now := time.Now().UTC()
	clock := fixedClock{now: now}
	interviewID := uuid.New()
	idGen := fixedIDGen{id: interviewID}
	eventFactory := eventdomain.NewFactory(idGen)

	svc := NewService(itRepo, appChecker, confSvc, outboxRepo, authPort, txMgr, eventFactory, clock, idGen)

	appID := uuid.New()
	actorID := uuid.New()
	interviewerID := uuid.New()

	startTime := now.Add(2 * time.Hour)
	endTime := startTime.Add(time.Hour)

	// 1. Schedule Interview
	id, err := svc.ScheduleInterview(ctx, ScheduleInterviewInput{
		ApplicationID:      appID,
		InterviewType:      "TECHNICAL",
		ScheduledStartTime: startTime,
		ScheduledEndTime:   endTime,
		Participants: []interviewdomain.Participant{
			{UserID: interviewerID, Role: "tech_interviewer"},
		},
		ActorID:           actorID,
		RequireOnlineRoom: true,
	})
	require.NoError(t, err)
	require.Equal(t, interviewID, id)
	require.Len(t, confSvc.createdRooms, 1)
	require.Len(t, outboxRepo.messages, 1)

	// 2. Submit Feedback
	err = svc.SubmitFeedback(ctx, SubmitFeedbackInput{
		InterviewID:       interviewID,
		InterviewerUserID: interviewerID,
		Score:             5,
		Notes:             "Exceeded expectations in algorithms and design",
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 2)

	// 3. Complete Interview
	err = svc.CompleteInterview(ctx, CompleteInterviewInput{
		InterviewID: interviewID,
		ActorID:     actorID,
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 3)

	// 4. Query Interview
	it, err := svc.GetInterview(ctx, interviewID, actorID)
	require.NoError(t, err)
	require.Equal(t, interviewdomain.StatusCompleted, it.Status())
	require.Len(t, it.Feedbacks(), 1)
}

