package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	"github.com/stretchr/testify/require"
)

func TestCreatedEventContainsAuthenticationSnapshot(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	payload := CreatedEvent{
		SessionID:            uuid.New(),
		IdentityID:           uuid.New(),
		OAuthClientID:        uuid.New(),
		AuthenticationLevel:  authenticationdomain.LevelMultiFactor,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP},
		Scopes:               []oauthdomain.Scope{"openid", "profile"},
		AuthenticatedAt:      now,
		IdleExpiresAt:        now.Add(time.Hour),
		AbsoluteExpiresAt:    now.Add(24 * time.Hour),
		CreatedAt:            now,
	}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))
	for _, field := range []string{
		"session_id",
		"identity_id",
		"oauth_client_id",
		"authentication_level",
		"authenticated_methods",
		"scopes",
		"authenticated_at",
		"idle_expires_at",
		"absolute_expires_at",
		"created_at",
	} {
		require.Contains(t, fields, field)
	}
}
