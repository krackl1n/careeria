package oauth

import (
	"errors"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func invalidTokenRequest(cause error) error {
	return apperr.New(apperr.TypeInvalidArgument, "oauth_invalid_request", "OAuth token request is invalid", cause)
}

func invalidRevocationRequest(cause error) error {
	return apperr.New(apperr.TypeInvalidArgument, "oauth_invalid_request", "OAuth token revocation request is invalid", cause)
}

func invalidClient(cause error) error {
	return apperr.New(apperr.TypeUnauthorized, "oauth_invalid_client", "OAuth client authentication failed", cause)
}

func invalidGrant(cause error) error {
	return apperr.New(apperr.TypeInvalidArgument, "oauth_invalid_grant", "OAuth grant is invalid", cause)
}

func invalidAccessToken(cause error) error {
	return apperr.New(apperr.TypeUnauthorized, "oauth_invalid_token", "access token is invalid", cause)
}

func insufficientScope(cause error) error {
	return apperr.New(apperr.TypeForbidden, "oauth_insufficient_scope", "access token has insufficient scope", cause)
}

func clientGrantError(cause error) error {
	if errors.Is(cause, oauthclientdomain.ErrInvalidStatus) {
		return invalidClient(cause)
	}
	return apperr.New(
		apperr.TypeForbidden,
		"oauth_unauthorized_client",
		"OAuth client is not allowed to use this grant type",
		cause,
	)
}
