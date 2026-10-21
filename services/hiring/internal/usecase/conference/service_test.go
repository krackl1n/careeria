package conference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockLiveKitClient struct {
	created string
	deleted string
}

func (m *mockLiveKitClient) CreateRoom(_ context.Context, name string) (string, error) {
	m.created = name
	return "https://livekit.test/" + name, nil
}

func (m *mockLiveKitClient) DeleteRoom(_ context.Context, name string) error {
	m.deleted = name
	return nil
}

func TestConferenceService(t *testing.T) {
	ctx := context.Background()
	mock := &mockLiveKitClient{}
	svc := NewService(mock)

	url, err := svc.CreateRoom(ctx, "interview-123")
	require.NoError(t, err)
	require.Equal(t, "https://livekit.test/interview-123", url)
	require.Equal(t, "interview-123", mock.created)

	err = svc.EndRoom(ctx, "interview-123")
	require.NoError(t, err)
	require.Equal(t, "interview-123", mock.deleted)
}

