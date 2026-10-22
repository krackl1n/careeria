package room

import (
	"context"
	"fmt"
	"strings"

	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type ConferenceClient interface {
	CreateRoom(ctx context.Context, roomName string) (string, error)
	EndRoom(ctx context.Context, roomName string) error
}

type LiveKitStrategy struct {
	client ConferenceClient
}

func NewLiveKitStrategy(client ConferenceClient) *LiveKitStrategy {
	return &LiveKitStrategy{client: client}
}

func (s *LiveKitStrategy) Format() interviewdomain.MeetingFormat {
	return interviewdomain.FormatOnlineLiveKit
}

func (s *LiveKitStrategy) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	roomName := fmt.Sprintf("interview-%s", req.InterviewID)
	url := fmt.Sprintf("https://meet.careeria.internal/rooms/%s", roomName)

	if s.client != nil {
		createdURL, err := s.client.CreateRoom(ctx, roomName)
		if err != nil {
			return nil, fmt.Errorf("provision livekit room: %w", err)
		}
		if createdURL != "" {
			url = createdURL
		}
	}

	return &ProvisionResult{
		MeetingURL: url,
		Metadata: map[string]string{
			"room_name": roomName,
			"provider":  "livekit",
		},
	}, nil
}

func (s *LiveKitStrategy) GenerateAccess(ctx context.Context, req AccessRequest) (*AccessResult, error) {
	roomName := fmt.Sprintf("interview-%s", req.InterviewID)
	token := fmt.Sprintf("token_%s_%s", roomName, req.UserID)
	joinURL := fmt.Sprintf("%s?token=%s", strings.TrimRight(req.MeetingURL, "/"), token)

	return &AccessResult{
		JoinURL:      joinURL,
		Token:        token,
		Instructions: "Please check your camera and microphone before joining.",
	}, nil
}

func (s *LiveKitStrategy) Teardown(ctx context.Context, req TeardownRequest) error {
	if s.client == nil {
		return nil
	}

	roomName := fmt.Sprintf("interview-%s", req.InterviewID)
	if err := s.client.EndRoom(ctx, roomName); err != nil {
		return fmt.Errorf("teardown livekit room: %w", err)
	}

	return nil
}
