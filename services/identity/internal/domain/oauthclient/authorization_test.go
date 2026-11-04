package oauthclient

import (
	"testing"
	"time"

	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	"github.com/stretchr/testify/require"
)

func TestValidateAuthorizationCodeFlow(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newPublicClient(t, now)
	redirectURI := client.RedirectURIs[0].URI
	scope := mustScope(t, "openid")
	require.NoError(t, client.AddScope(scope, now.Add(time.Minute)))

	require.NoError(t, client.ValidateAuthorizationRedirectURI(redirectURI))
	require.NoError(t, client.ValidateAuthorizationCodeFlow(
		[]oauthdomain.Scope{scope},
		now.Add(2*time.Minute),
	))
}

func TestValidateAuthorizationRedirectURI(t *testing.T) {
	t.Parallel()

	client := newPublicClient(t, testTime())

	require.NoError(t, client.ValidateAuthorizationRedirectURI(client.RedirectURIs[0].URI))
	require.ErrorIs(
		t,
		client.ValidateAuthorizationRedirectURI("https://other.example/callback"),
		ErrRedirectURINotFound,
	)
}

func TestValidateAuthorizationCodeFlowRejectsClientInvariantViolations(t *testing.T) {
	t.Parallel()

	now := testTime()

	tests := []struct {
		name    string
		prepare func(*OAuthClient)
		scopes  []oauthdomain.Scope
		wantErr error
	}{
		{
			name: "inactive client",
			prepare: func(client *OAuthClient) {
				client.Status = StatusDisabled
			},
			wantErr: ErrInvalidStatus,
		},
		{
			name: "authorization code grant is missing",
			prepare: func(client *OAuthClient) {
				client.GrantTypes = nil
			},
			wantErr: ErrAuthorizationCodeFlowNotAllowed,
		},
		{
			name:    "scope is not registered",
			scopes:  []oauthdomain.Scope{"admin"},
			wantErr: ErrScopeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newPublicClient(t, now)
			scope := mustScope(t, "openid")
			require.NoError(t, client.AddScope(scope, now.Add(time.Minute)))
			if tt.prepare != nil {
				tt.prepare(client)
			}

			scopes := tt.scopes
			if scopes == nil {
				scopes = []oauthdomain.Scope{scope}
			}

			err := client.ValidateAuthorizationCodeFlow(
				scopes,
				now.Add(2*time.Minute),
			)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
