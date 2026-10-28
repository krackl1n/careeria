package accesstoken

import (
	"errors"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

var ErrInvalidToken = errors.New("access token is invalid")

type VerifiedToken struct {
	ID         uuid.UUID
	IdentityID identitydomain.ID
	SessionID  sessiondomain.ID
	ClientID   oauthclientdomain.ClientID
	Scopes     []oauthdomain.Scope
}
