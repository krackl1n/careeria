package oauthclient

import (
	"errors"
	"time"
)

type PostLogoutRedirectURI struct {
	ID  PostLogoutRedirectURIID
	URI PostLogoutRedirectURIValue

	CreatedAt time.Time
}

func newPostLogoutRedirectURI(
	id PostLogoutRedirectURIID,
	uri PostLogoutRedirectURIValue,
	now time.Time,
) (PostLogoutRedirectURI, error) {
	if !id.IsValid() {
		return PostLogoutRedirectURI{}, errors.New("identifier is required")
	}
	if now.IsZero() {
		return PostLogoutRedirectURI{}, errors.New("creation time is invalid")
	}

	return PostLogoutRedirectURI{ID: id, URI: uri, CreatedAt: now}, nil
}
