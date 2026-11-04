package idtoken

import "time"

type Claims struct {
	Issuer   string
	Subject  string
	Audience string

	ExpiresAt time.Time
	IssuedAt  time.Time

	Nonce *string

	Identity IdentityClaims

	AuthenticationLevel  string
	AuthenticatedMethods []string
	AuthenticatedAt      time.Time
}

type IdentityClaims struct {
	Email         *string
	EmailVerified *bool

	PhoneNumber         *string
	PhoneNumberVerified *bool

	Name       *string
	FamilyName *string
	GivenName  *string
	MiddleName *string
	UpdatedAt  *time.Time
}
