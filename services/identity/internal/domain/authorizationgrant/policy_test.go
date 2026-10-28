package authorizationgrant

import (
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestResolveConsentUsesRememberedGrant(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	client := consentClientForTest()
	grant, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		client.ID,
		SourceUserConsent,
		[]oauthdomain.Scope{"openid", "profile"},
		nil,
		now,
	)
	require.NoError(t, err)

	decision, err := ResolveConsent(client, grant, []oauthdomain.Scope{"openid"}, false, now)
	require.NoError(t, err)
	require.Equal(t, ConsentRememberedGrant, decision.Kind)
}

func TestResolveConsentSkipsTrustedClient(t *testing.T) {
	t.Parallel()

	client := consentClientForTest()
	client.ConsentMode = oauthclientdomain.ConsentModeTrusted
	decision, err := ResolveConsent(
		client,
		nil,
		[]oauthdomain.Scope{"openid"},
		false,
		time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, ConsentTrustedClient, decision.Kind)
}

func TestResolveConsentCanBeForced(t *testing.T) {
	t.Parallel()

	client := consentClientForTest()
	client.ConsentMode = oauthclientdomain.ConsentModeTrusted
	decision, err := ResolveConsent(
		client,
		nil,
		[]oauthdomain.Scope{"openid"},
		true,
		time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, ConsentRequired, decision.Kind)
}

func consentClientForTest() *oauthclientdomain.OAuthClient {
	return &oauthclientdomain.OAuthClient{
		ID:          oauthclientdomain.ID(uuid.New()),
		ConsentMode: oauthclientdomain.ConsentModeRemember,
		Scopes: []oauthclientdomain.ScopeRegistration{
			{Scope: "openid"},
			{Scope: "profile"},
		},
	}
}
