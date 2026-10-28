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

func TestAuthorizationGrantCoversRememberedScopes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	grant, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		SourceUserConsent,
		[]oauthdomain.Scope{"openid", "profile"},
		nil,
		now,
	)
	require.NoError(t, err)

	require.True(t, grant.Covers([]oauthdomain.Scope{"openid"}, now))
	require.False(t, grant.Covers([]oauthdomain.Scope{"email"}, now))
}

func TestAuthorizationGrantCanBeExtended(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	grant, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		SourceUserConsent,
		[]oauthdomain.Scope{"openid"},
		nil,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, grant.Extend(
		[]oauthdomain.Scope{"openid", "email"},
		nil,
		now.Add(time.Minute),
	))
	require.Equal(t, []oauthdomain.Scope{"openid", "email"}, grant.Scopes)
}

func TestRevokedAuthorizationGrantDoesNotCoverScopes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	grant, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		SourceUserConsent,
		[]oauthdomain.Scope{"openid"},
		nil,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, grant.Revoke(now.Add(time.Minute)))
	require.False(t, grant.Covers([]oauthdomain.Scope{"openid"}, now.Add(2*time.Minute)))
}

func TestAuthorizationGrantCanBeReplacedAfterRevocation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	grant, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		SourceUserConsent,
		[]oauthdomain.Scope{"openid"},
		nil,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, grant.Revoke(now.Add(time.Minute)))

	regrantedAt := now.Add(2 * time.Minute)
	require.NoError(t, grant.Replace(
		SourceUserConsent,
		[]oauthdomain.Scope{"openid", "email"},
		nil,
		regrantedAt,
	))
	require.Nil(t, grant.RevokedAt)
	require.Equal(t, regrantedAt, grant.GrantedAt)
	require.True(t, grant.Covers([]oauthdomain.Scope{"email"}, regrantedAt))
}
