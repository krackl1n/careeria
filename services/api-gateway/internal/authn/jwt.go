package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	sharedjwt "github.com/krackl1n/careeria/jwt"
)

type PublicKeyResolver func(context.Context, string) ([]byte, error)

type JWTAuthenticator struct {
	config  AuthConfig
	resolve PublicKeyResolver
}

func NewJWT(config AuthConfig, resolve PublicKeyResolver) (*JWTAuthenticator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if resolve == nil {
		return nil, errors.New("JWT public key resolver is required")
	}
	return &JWTAuthenticator{config: config, resolve: resolve}, nil
}

func (a *JWTAuthenticator) Authenticate(request *http.Request) (ctxmeta.Principal, error) {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return ctxmeta.Principal{}, errors.New("Bearer access token is required")
	}

	keyID, err := sharedjwt.ParseKeyID(parts[1])
	if err != nil {
		return ctxmeta.Principal{}, fmt.Errorf("parse access token key id: %w", err)
	}

	publicJWK, err := a.resolve(request.Context(), keyID)
	if err != nil {
		return ctxmeta.Principal{}, fmt.Errorf("resolve access token public key: %w", err)
	}

	claims, err := sharedjwt.VerifyAccessToken(
		sharedjwt.VerifyAccessTokenInput{
			Value:     parts[1],
			KeyID:     keyID,
			Algorithm: sharedjwt.AlgorithmEdDSA,
			PublicJWK: publicJWK,
			Issuer:    a.config.Issuer,
			Audience:  a.config.Audience,
			Now:       time.Now().UTC(),
		},
	)
	if err != nil {
		return ctxmeta.Principal{}, fmt.Errorf("verify access token: %w", err)
	}

	return ctxmeta.Principal{
		SubjectID: claims.Subject,
	}, nil
}
