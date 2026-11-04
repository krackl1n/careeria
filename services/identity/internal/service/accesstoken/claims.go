package accesstoken

import "time"

type Claims struct {
	Issuer   string
	Subject  string
	Audience []string

	ExpiresAt time.Time
	IssuedAt  time.Time
	TokenID   string

	ClientID  string
	Scopes    []string
	SessionID string

	AuthenticationLevel  string
	AuthenticatedMethods []string
	AuthenticatedAt      time.Time
}
