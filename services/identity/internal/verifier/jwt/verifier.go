package jwt

import (
	"fmt"

	sharedjwt "github.com/krackl1n/careeria/jwt"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
)

type Verifier struct{}

func (Verifier) KeyID(value string) (signingkeydomain.KeyID, error) {
	keyID, err := sharedjwt.ParseKeyID(value)
	if err != nil {
		return "", err
	}

	parsed, err := signingkeydomain.NewKeyID(keyID)
	if err != nil {
		return "", fmt.Errorf("decode JWT key ID: %w", err)
	}
	return parsed, nil
}

func (Verifier) Verify(input accesstokenservice.VerifyInput) (accesstokenservice.Claims, error) {
	claims, err := sharedjwt.VerifyAccessToken(sharedjwt.VerifyAccessTokenInput{
		Value:     input.Value,
		KeyID:     string(input.KeyID),
		Algorithm: string(input.Algorithm),
		PublicJWK: []byte(input.PublicJWK),
		Issuer:    input.Issuer,
		Audience:  input.Audience,
		Now:       input.Now,
	})
	if err != nil {
		return accesstokenservice.Claims{}, err
	}

	return accesstokenservice.Claims(claims), nil
}
