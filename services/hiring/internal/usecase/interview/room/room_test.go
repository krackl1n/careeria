package room

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
	"github.com/stretchr/testify/require"
)

type mockConferenceClient struct {
	createdRooms []string
	endedRooms   []string
}

func (m *mockConferenceClient) CreateRoom(_ context.Context, roomName string) (string, error) {
	m.createdRooms = append(m.createdRooms, roomName)
	return "https://livekit.custom/" + roomName, nil
}

func (m *mockConferenceClient) EndRoom(_ context.Context, roomName string) error {
	m.endedRooms = append(m.endedRooms, roomName)
	return nil
}

func TestMeetingRoomStrategiesAndRegistry(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockConferenceClient{}

	livekitStrategy := NewLiveKitStrategy(mockClient)
	onsiteStrategy := NewOnsiteStrategy("Main Office, 4th Floor")

	roomStrategy := NewRoomStrategy(livekitStrategy, onsiteStrategy)

	interviewID := uuid.New()
	appID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	t.Run("LiveKit strategy lifecycle via RoomStrategy", func(t *testing.T) {
		res, err := roomStrategy.Provision(ctx, interviewdomain.FormatOnlineLiveKit, ProvisionRequest{
			InterviewID:   interviewID,
			ApplicationID: appID,
			InterviewType: interviewdomain.TypeSystemDesign,
			StartTime:     now,
			EndTime:       now.Add(time.Hour),
		})
		require.NoError(t, err)
		require.Equal(t, "https://livekit.custom/interview-"+interviewID.String(), res.MeetingURL)
		require.Len(t, mockClient.createdRooms, 1)

		access, err := roomStrategy.GenerateAccess(ctx, interviewdomain.FormatOnlineLiveKit, AccessRequest{
			InterviewID: interviewID,
			UserID:      userID,
			UserRole:    interviewdomain.RoleCandidate,
			MeetingURL:  res.MeetingURL,
		})
		require.NoError(t, err)
		require.NotEmpty(t, access.JoinURL)
		require.NotEmpty(t, access.Token)

		err = roomStrategy.Teardown(ctx, interviewdomain.FormatOnlineLiveKit, TeardownRequest{
			InterviewID: interviewID,
			MeetingURL:  res.MeetingURL,
		})
		require.NoError(t, err)
		require.Len(t, mockClient.endedRooms, 1)
	})

	t.Run("Onsite strategy lifecycle via RoomStrategy", func(t *testing.T) {
		res, err := roomStrategy.Provision(ctx, interviewdomain.FormatOnsite, ProvisionRequest{
			InterviewID:   interviewID,
			ApplicationID: appID,
			InterviewType: interviewdomain.TypeBehavioral,
			Location:      "Conference Room Alpha, 2nd floor",
		})
		require.NoError(t, err)
		require.Equal(t, "Conference Room Alpha, 2nd floor", res.MeetingURL)

		access, err := roomStrategy.GenerateAccess(ctx, interviewdomain.FormatOnsite, AccessRequest{
			InterviewID: interviewID,
			UserID:      userID,
			UserRole:    interviewdomain.RoleCandidate,
			MeetingURL:  res.MeetingURL,
		})
		require.NoError(t, err)
		require.Contains(t, access.Instructions, "Conference Room Alpha")

		err = roomStrategy.Teardown(ctx, interviewdomain.FormatOnsite, TeardownRequest{
			InterviewID: interviewID,
			MeetingURL:  res.MeetingURL,
		})
		require.NoError(t, err)
	})

	t.Run("Onsite fallback to default address", func(t *testing.T) {
		res, err := onsiteStrategy.Provision(ctx, ProvisionRequest{
			InterviewID: interviewID,
		})
		require.NoError(t, err)
		require.Equal(t, "Main Office, 4th Floor", res.MeetingURL)
	})

	t.Run("Unsupported format error", func(t *testing.T) {
		_, err := roomStrategy.Resolve(interviewdomain.MeetingFormat("UNKNOWN"))
		require.ErrorIs(t, err, ErrUnsupportedFormat)

		_, err = roomStrategy.Provision(ctx, interviewdomain.MeetingFormat("UNKNOWN"), ProvisionRequest{})
		require.ErrorIs(t, err, ErrUnsupportedFormat)
	})
}
