package oauthclient

import (
	"time"
)

type GrantTypeRegistration struct {
	GrantType GrantType

	CreatedAt time.Time
}

func newGrantTypeRegistration(grantType GrantType, now time.Time) GrantTypeRegistration {
	return GrantTypeRegistration{GrantType: grantType, CreatedAt: now}
}
