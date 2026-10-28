package jwt

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const AccessTokenType = "at+jwt"
const AlgorithmEdDSA = "EdDSA"

type AccessTokenClaims struct {
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

type VerifyAccessTokenInput struct {
	Value     string
	KeyID     string
	Algorithm string
	PublicJWK []byte
	Issuer    string
	Audience  []string
	Now       time.Time
}

func ParseKeyID(value string) (string, error) {
	token, _, err := new(jwtlib.Parser).ParseUnverified(value, &accessTokenClaims{})
	if err != nil {
		return "", fmt.Errorf("decode JWT header: %w", err)
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok || strings.TrimSpace(keyID) == "" {
		return "", errors.New("JWT key ID is required")
	}

	return keyID, nil
}

func VerifyAccessToken(input VerifyAccessTokenInput) (AccessTokenClaims, error) {
	if input.Algorithm != AlgorithmEdDSA {
		return AccessTokenClaims{}, errors.New("JWT verification algorithm is not supported")
	}

	publicKey, err := ed25519PublicKey(input.PublicJWK, input.KeyID, input.Algorithm)
	if err != nil {
		return AccessTokenClaims{}, fmt.Errorf("decode JWT public key: %w", err)
	}

	claims := &accessTokenClaims{}
	token, err := jwtlib.ParseWithClaims(
		input.Value,
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Header["typ"] != AccessTokenType {
				return nil, errors.New("JWT type is invalid")
			}
			if token.Header["kid"] != input.KeyID {
				return nil, errors.New("JWT key ID does not match signing key")
			}
			return publicKey, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
		jwtlib.WithIssuer(input.Issuer),
		jwtlib.WithAllAudiences(input.Audience...),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithIssuedAt(),
		jwtlib.WithTimeFunc(func() time.Time { return input.Now }),
		jwtlib.WithStrictDecoding(),
	)
	if err != nil {
		return AccessTokenClaims{}, fmt.Errorf("verify JWT: %w", err)
	}
	if !token.Valid {
		return AccessTokenClaims{}, errors.New("JWT is invalid")
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		return AccessTokenClaims{}, errors.New("JWT timestamps are invalid")
	}

	return AccessTokenClaims{
		Issuer:               claims.Issuer,
		Subject:              claims.Subject,
		Audience:             append([]string(nil), claims.Audience...),
		ExpiresAt:            claims.ExpiresAt.Time.UTC(),
		IssuedAt:             claims.IssuedAt.Time.UTC(),
		TokenID:              claims.ID,
		ClientID:             claims.ClientID,
		Scopes:               strings.Fields(claims.Scopes),
		SessionID:            claims.SessionID,
		AuthenticationLevel:  claims.AuthenticationLevel,
		AuthenticatedMethods: append([]string(nil), claims.AuthenticatedMethods...),
		AuthenticatedAt:      numericDateTime(claims.AuthenticatedAt),
	}, nil
}

type accessTokenClaims struct {
	ClientID  string `json:"client_id"`
	Scopes    string `json:"scope"`
	SessionID string `json:"sid"`

	AuthenticationLevel  string              `json:"acr"`
	AuthenticatedMethods []string            `json:"amr"`
	AuthenticatedAt      *jwtlib.NumericDate `json:"auth_time"`

	jwtlib.RegisteredClaims
}

type publicJWK struct {
	KeyType   string `json:"kty"`
	Curve     string `json:"crv"`
	X         string `json:"x"`
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
}

func ed25519PublicKey(value []byte, keyID, algorithm string) (ed25519.PublicKey, error) {
	var jwk publicJWK
	if err := json.Unmarshal(value, &jwk); err != nil {
		return nil, fmt.Errorf("decode JWK: %w", err)
	}
	if jwk.KeyType != "OKP" || jwk.Curve != "Ed25519" {
		return nil, errors.New("JWK must contain an Ed25519 public key")
	}
	if jwk.KeyID != keyID ||
		jwk.Algorithm != "" && jwk.Algorithm != algorithm ||
		jwk.Use != "" && jwk.Use != "sig" {
		return nil, errors.New("JWK metadata does not match signing key")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil || len(decoded) != ed25519.PublicKeySize || base64.RawURLEncoding.EncodeToString(decoded) != jwk.X {
		return nil, errors.New("JWK public key is invalid")
	}

	return ed25519.PublicKey(decoded), nil
}

func numericDateTime(value *jwtlib.NumericDate) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.Time.UTC()
}
