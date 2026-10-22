package interview

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/krackl1n/careeria/services/hiring/internal/usecase/interview/room"
	"github.com/stretchr/testify/require"
)

type mockInterviewRepo struct {
	interviews map[uuid.UUID]*interviewdomain.Interview
}

func newMockInterviewRepo() *mockInterviewRepo {
	return &mockInterviewRepo{interviews: make(map[uuid.UUID]*interviewdomain.Interview)}
}

func (m *mockInterviewRepo) Create(_ context.Context, it *interviewdomain.Interview) error {
	m.interviews[it.ID] = it
	return nil
}

func (m *mockInterviewRepo) Update(_ context.Context, it *interviewdomain.Interview) error {
	m.interviews[it.ID] = it
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
		if it.ApplicationID == appID {
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

type mockConferenceClient struct {
	createdRooms []string
	endedRooms   []string
}

func (m *mockConferenceClient) CreateRoom(_ context.Context, roomName string) (string, error) {
	m.createdRooms = append(m.createdRooms, roomName)
	return "https://livekit.example/rooms/" + roomName, nil
}

func (m *mockConferenceClient) EndRoom(_ context.Context, roomName string) error {
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

func TestInterviewService(t *testing.T) {
	ctx := context.Background()
	itRepo := newMockInterviewRepo()
	appChecker := &mockAppStageChecker{stage: applicationdomain.StageInterview}
	confClient := &mockConferenceClient{}

	roomStrategy := room.NewRoomStrategy(
		room.NewLiveKitStrategy(confClient),
		room.NewOnsiteStrategy("Main Office, 4th floor"),
	)

	outboxRepo := &mockOutboxRepo{}
	authPort := &mockAuthPort{}
	txMgr := &mockTxManager{}

	now := time.Now().UTC()
	clock := fixedClock{now: now}
	interviewID1 := uuid.New()
	interviewID2 := uuid.New()
	idGen := &sequentialIDGen{ids: []uuid.UUID{interviewID1, interviewID2}}
	eventFactory := eventdomain.NewFactory(idGen)

	svc := NewService(itRepo, appChecker, roomStrategy, outboxRepo, authPort, txMgr, eventFactory, clock, idGen)

	appID := uuid.New()
	actorID := uuid.New()
	interviewerID := uuid.New()

	startTime := now.Add(2 * time.Hour)
	endTime := startTime.Add(time.Hour)

	pInterviewer, err := interviewdomain.NewParticipant(interviewerID, interviewdomain.RoleInterviewer)
	require.NoError(t, err)

	id1, err := svc.ScheduleInterview(ctx, ScheduleInterviewInput{
		ApplicationID:      appID,
		InterviewType:      interviewdomain.TypeTechnical,
		Format:             interviewdomain.FormatOnlineLiveKit,
		ScheduledStartTime: startTime,
		ScheduledEndTime:   endTime,
		Participants:       []interviewdomain.Participant{pInterviewer},
		ActorID:            actorID,
	})
	require.NoError(t, err)
	require.Equal(t, interviewID1, id1)
	require.Len(t, confClient.createdRooms, 1)
	require.Len(t, outboxRepo.messages, 1)

	access, err := svc.GetMeetingAccess(ctx, GetMeetingAccessInput{
		InterviewID: interviewID1,
		ActorID:     interviewerID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, access.JoinURL)
	require.NotEmpty(t, access.Token)

	err = svc.SubmitFeedback(ctx, SubmitFeedbackInput{
		InterviewID:       interviewID1,
		InterviewerUserID: interviewerID,
		Score:             5,
		Notes:             "Exceeded expectations in algorithms and design",
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 2)

	err = svc.CompleteInterview(ctx, CompleteInterviewInput{
		InterviewID: interviewID1,
		ActorID:     actorID,
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 3)

	it, err := svc.GetInterview(ctx, interviewID1, actorID)
	require.NoError(t, err)
	require.Equal(t, interviewdomain.StatusCompleted, it.Status)
	require.Equal(t, interviewdomain.FormatOnlineLiveKit, it.Format)
	require.Len(t, it.Feedbacks, 1)

	pHr, err := interviewdomain.NewParticipant(interviewerID, interviewdomain.RoleRecruiter)
	require.NoError(t, err)

	id2, err := svc.ScheduleInterview(ctx, ScheduleInterviewInput{
		ApplicationID:      appID,
		InterviewType:      interviewdomain.TypeBehavioral,
		Format:             interviewdomain.FormatOnsite,
		ScheduledStartTime: startTime.Add(24 * time.Hour),
		ScheduledEndTime:   endTime.Add(24 * time.Hour),
		Location:           "Room 302, Green Wing",
		Participants:       []interviewdomain.Participant{pHr},
		ActorID:            actorID,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id2)

	onsiteIt, err := svc.GetInterview(ctx, id2, actorID)
	require.NoError(t, err)
	require.Equal(t, interviewdomain.FormatOnsite, onsiteIt.Format)
	require.Equal(t, "Room 302, Green Wing", onsiteIt.MeetingRoomURL)

	onsiteAccess, err := svc.GetMeetingAccess(ctx, GetMeetingAccessInput{
		InterviewID: id2,
		ActorID:     interviewerID,
	})
	require.NoError(t, err)
	require.Contains(t, onsiteAccess.Instructions, "Room 302, Green Wing")

	err = svc.CancelInterview(ctx, CancelInterviewInput{
		InterviewID: id2,
		Reason:      "Rescheduled by recruiter",
		ActorID:     actorID,
	})
	require.NoError(t, err)
	cancelledIt, err := svc.GetInterview(ctx, id2, actorID)
	require.NoError(t, err)
	require.Equal(t, interviewdomain.StatusCancelled, cancelledIt.Status)
}
