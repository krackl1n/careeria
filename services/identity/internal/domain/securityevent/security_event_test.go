package securityevent

import (
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	serviceaccountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityEvent(t *testing.T) {
	t.Parallel()

	identityID := identitydomain.ID(uuid.New())
	actor, err := NewIdentityActor(identityID)
	require.NoError(t, err)
	eventType, err := NewEventType("authentication.succeeded")
	require.NoError(t, err)
	ipAddress, err := sessiondomain.NewIPAddress("192.0.2.10")
	require.NoError(t, err)
	metadataBytes := []byte(`{"authentication_method":"password"}`)
	metadata, err := NewMetadata(metadataBytes)
	require.NoError(t, err)
	identifierBytes := []byte("identifier-hash")
	identifierHash, err := NewIdentifierHash(identifierBytes)
	require.NoError(t, err)
	sessionID := sessiondomain.ID(uuid.New())
	oauthClientID := oauthclientdomain.ID(uuid.New())
	correlationID := uuid.New()
	userAgent := "browser/1.0"

	event, err := New(
		ID(uuid.New()),
		eventType,
		ResultSuccess,
		actor,
		&sessionID,
		&oauthClientID,
		identifierHash,
		&ipAddress,
		&userAgent,
		&correlationID,
		metadata,
		securityEventTestTime(),
	)
	require.NoError(t, err)
	require.Equal(t, identityID, *event.Actor.IdentityID)
	require.Equal(t, sessionID, *event.SessionID)
	require.Equal(t, oauthClientID, *event.OAuthClientID)
	require.JSONEq(t, string(metadata), string(event.Metadata))

	metadataBytes[0] = '['
	metadata[0] = '['
	identifierBytes[0] = 'x'
	identifierHash[0] = 'y'
	require.Equal(t, byte('{'), event.Metadata[0])
	require.Equal(t, byte('i'), event.IdentifierHash[0])
}

func TestSecurityEventActors(t *testing.T) {
	t.Parallel()

	identityID := identitydomain.ID(uuid.New())
	identityActor, err := NewIdentityActor(identityID)
	require.NoError(t, err)
	require.Equal(t, ActorTypeIdentity, identityActor.Type)
	require.Equal(t, identityID, *identityActor.IdentityID)

	administrator, err := NewAdministratorActor(identityID)
	require.NoError(t, err)
	require.Equal(t, ActorTypeAdministrator, administrator.Type)

	serviceAccountID := serviceaccountdomain.ID(uuid.New())
	serviceAccount, err := NewServiceAccountActor(serviceAccountID)
	require.NoError(t, err)
	require.Equal(t, serviceAccountID, *serviceAccount.ServiceAccountID)

	require.Equal(t, ActorTypeAnonymous, NewAnonymousActor().Type)
	require.Equal(t, ActorTypeSystem, NewSystemActor().Type)
}

func TestSecurityEventActorConstructorsRejectInvalidIdentifiers(t *testing.T) {
	t.Parallel()

	_, err := NewIdentityActor(identitydomain.ID(uuid.Nil))
	require.Error(t, err)
	_, err = NewServiceAccountActor(serviceaccountdomain.ID(uuid.Nil))
	require.Error(t, err)
}

func TestSecurityEventMetadataDefaultsToEmptyObject(t *testing.T) {
	t.Parallel()

	metadata, err := NewMetadata(nil)
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(metadata))

	_, err = NewMetadata([]byte(`[]`))
	require.Error(t, err)
}

func TestSecurityEventRejectsInvalidOptionalReferences(t *testing.T) {
	t.Parallel()

	eventType, err := NewEventType("authentication.failed")
	require.NoError(t, err)
	metadata, err := NewMetadata(nil)
	require.NoError(t, err)
	nilSessionID := sessiondomain.ID(uuid.Nil)

	_, err = New(
		ID(uuid.New()),
		eventType,
		ResultFailure,
		NewAnonymousActor(),
		&nilSessionID,
		nil,
		nil,
		nil,
		nil,
		nil,
		metadata,
		securityEventTestTime(),
	)
	require.Error(t, err)
}

func securityEventTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
