package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

// todo Переделать, мне это не нравится. МОжет в домен перенести или в app сервис если есть оркестрация
type securityEventInput struct {
	Type          securityeventdomain.EventType
	Result        securityeventdomain.Result
	Actor         securityeventdomain.Actor
	SessionID     *sessiondomain.ID
	OAuthClientID *oauthclientdomain.ID
	Metadata      map[string]any
	OccurredAt    time.Time
}

func securityFailureReason(err error) string {
	if value, ok := apperr.As(err); ok {
		return value.Code()
	}
	return "operation_failed"
}

func isOAuthFailure(err error) bool {
	value, ok := apperr.As(err)
	return ok && value.Code() != "internal_error"
}

func (s *Service) saveSecurityEvent(ctx context.Context, input securityEventInput) error {
	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return fmt.Errorf("marshal %s security event metadata: %w", input.Type, err)
	}
	parsedMetadata, err := securityeventdomain.NewMetadata(metadata)
	if err != nil {
		return fmt.Errorf("create %s security event metadata: %w", input.Type, err)
	}
	event, err := securityeventdomain.New(
		securityeventdomain.ID(s.ids.New()),
		input.Type,
		input.Result,
		input.Actor,
		input.SessionID,
		input.OAuthClientID,
		nil, //  todo Тут тоже изменить и не переносить так
		nil,
		nil,
		nil,
		parsedMetadata,
		input.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("create %s security event: %w", input.Type, err)
	}
	if err := s.securityEvents.Create(ctx, event); err != nil {
		return fmt.Errorf("save %s security event: %w", input.Type, err)
	}
	return nil
}
