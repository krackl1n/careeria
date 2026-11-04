package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestNewSessionCopiesAuthenticationSnapshot(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	ipAddress, err := NewIPAddress("2001:0db8::1")
	require.NoError(t, err)
	userAgent := "  browser/1.0  "
	deviceName := "  Work laptop  "
	methods := []authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP}
	scopes := []oauthdomain.Scope{"openid", "profile"}

	entity, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		authenticationdomain.LevelMultiFactor,
		methods,
		scopes,
		&ipAddress,
		&userAgent,
		&deviceName,
		now,
		now.Add(30*time.Minute),
		now.Add(24*time.Hour),
		now,
	)
	require.NoError(t, err)

	methods[0] = authenticationdomain.MethodWebAuthn
	scopes[0] = "email"
	require.Equal(t, []authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP}, entity.AuthenticatedMethods)
	require.Equal(t, []oauthdomain.Scope{"openid", "profile"}, entity.Scopes)
	require.Equal(t, IPAddress("2001:db8::1"), *entity.IPAddress)
	require.Equal(t, "browser/1.0", *entity.UserAgent)
	require.Equal(t, "Work laptop", *entity.DeviceName)
	require.Equal(t, now, entity.CreatedAt)
	require.Equal(t, now, entity.AuthenticatedAt)
}

func TestNewSessionAcceptsAuthenticationBeforeOAuthSessionCreation(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	authenticatedAt := now.Add(-time.Minute)
	entity, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword},
		[]oauthdomain.Scope{"openid"},
		nil,
		nil,
		nil,
		authenticatedAt,
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		now,
	)

	require.NoError(t, err)
	require.Equal(t, authenticatedAt, entity.AuthenticatedAt)
	require.Equal(t, now, entity.CreatedAt)
}

func TestNewSessionRejectsInvalidAuthenticationSnapshot(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	_, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodPassword},
		nil,
		nil,
		nil,
		nil,
		now,
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		now,
	)
	require.Error(t, err)
}

func TestSessionRecordActivityExtendsIdleLifetime(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	entity := newSessionForTest(t, now, now.Add(15*time.Minute), now.Add(time.Hour))
	activityAt := now.Add(5 * time.Minute)
	extendedUntil := now.Add(25 * time.Minute)

	require.NoError(t, entity.RecordActivity(activityAt, extendedUntil))
	require.Equal(t, activityAt, *entity.LastSeenAt)
	require.Equal(t, extendedUntil, entity.IdleExpiresAt)
	require.NoError(t, entity.ValidateForUse(activityAt.Add(time.Minute)))
}

func TestSessionRecordActivityNeverShortensIdleLifetime(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	originalExpiration := now.Add(30 * time.Minute)
	entity := newSessionForTest(t, now, originalExpiration, now.Add(time.Hour))

	require.NoError(t, entity.RecordActivity(now.Add(5*time.Minute), now.Add(20*time.Minute)))
	require.Equal(t, originalExpiration, entity.IdleExpiresAt)
}

func TestSessionValidateForUseReturnsLifecycleErrors(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()

	idleExpired := newSessionForTest(t, now, now.Add(10*time.Minute), now.Add(time.Hour))
	require.ErrorIs(t, idleExpired.ValidateForUse(now.Add(10*time.Minute)), ErrIdleExpired)

	absoluteExpired := newSessionForTest(t, now, now.Add(time.Hour), now.Add(time.Hour))
	require.ErrorIs(t, absoluteExpired.ValidateForUse(now.Add(time.Hour)), ErrAbsoluteExpired)

	revoked := newSessionForTest(t, now, now.Add(time.Hour), now.Add(24*time.Hour))
	require.NoError(t, revoked.Revoke("identity_disabled", now.Add(time.Minute)))
	require.ErrorIs(t, revoked.ValidateForUse(now.Add(2*time.Minute)), ErrRevoked)
}

func TestSessionRevokeRequiresReasonAndIsIdempotent(t *testing.T) {
	t.Parallel()

	now := sessionTestTime()
	entity := newSessionForTest(t, now, now.Add(time.Hour), now.Add(24*time.Hour))

	require.Error(t, entity.Revoke(" ", now.Add(time.Minute)))
	require.Nil(t, entity.RevokedAt)

	revokedAt := now.Add(2 * time.Minute)
	require.NoError(t, entity.Revoke("  user_logout  ", revokedAt))
	require.Equal(t, revokedAt, *entity.RevokedAt)
	require.Equal(t, "user_logout", *entity.RevokeReason)
	require.NoError(t, entity.Revoke("different_reason", now.Add(3*time.Minute)))
	require.Equal(t, "user_logout", *entity.RevokeReason)
}

func newSessionForTest(t *testing.T, now, idleExpiresAt, absoluteExpiresAt time.Time) *Session {
	t.Helper()

	entity, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword},
		[]oauthdomain.Scope{"openid"},
		nil,
		nil,
		nil,
		now,
		idleExpiresAt,
		absoluteExpiresAt,
		now,
	)
	require.NoError(t, err)
	return entity
}

func sessionTestTime() time.Time {
	return time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
}
