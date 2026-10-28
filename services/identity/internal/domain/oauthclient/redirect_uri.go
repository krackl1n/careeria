package oauthclient

import (
	"errors"
	"time"
)

type RedirectURI struct {
	ID  RedirectURIID
	URI RedirectURIValue

	CreatedAt time.Time
}

func newRedirectURI(id RedirectURIID, uri RedirectURIValue, now time.Time) (RedirectURI, error) {
	if !id.IsValid() {
		return RedirectURI{}, errors.New("identifier is required")
	}
	if now.IsZero() {
		return RedirectURI{}, errors.New("creation time is invalid")
	}

	return RedirectURI{ID: id, URI: uri, CreatedAt: now}, nil
}
