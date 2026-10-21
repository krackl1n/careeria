package conference

import (
	"context"
	"fmt"
	"strings"
)

type LiveKitClient interface {
	CreateRoom(ctx context.Context, roomName string) (string, error)
	DeleteRoom(ctx context.Context, roomName string) error
}

type Service struct {
	client LiveKitClient
}

func NewService(client LiveKitClient) *Service {
	return &Service{client: client}
}

func (s *Service) CreateRoom(ctx context.Context, roomName string) (string, error) {
	if strings.TrimSpace(roomName) == "" {
		return "", fmt.Errorf("room name is required")
	}

	if s.client == nil {
		return fmt.Sprintf("https://livekit.careeria.internal/rooms/%s", roomName), nil
	}

	return s.client.CreateRoom(ctx, roomName)
}

func (s *Service) EndRoom(ctx context.Context, roomName string) error {
	if strings.TrimSpace(roomName) == "" {
		return fmt.Errorf("room name is required")
	}

	if s.client == nil {
		return nil
	}

	return s.client.DeleteRoom(ctx, roomName)
}

