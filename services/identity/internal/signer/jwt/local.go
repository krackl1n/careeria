package jwt

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
)

const accessTokenType = "at+jwt"
const idTokenType = "JWT"

type Signer struct {
	keyID         signingkey.KeyID
	privateKeyRef signingkey.PrivateKeyRef
	privateKey    ed25519.PrivateKey
	publicJWK     signingkey.PublicJWK
}

func NewSigner(cfg Config) (*Signer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate JWT signer config: %w", err)
	}

	privateKey, err := loadPrivateKey(cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load JWT signing key: %w", err)
	}
	keyID, err := signingkey.NewKeyID(cfg.KeyID)
	if err != nil {
		return nil, fmt.Errorf("parse JWT signing key ID: %w", err)
	}
	privateKeyRef, err := signingkey.NewPrivateKeyRef(cfg.PrivateKeyRef)
	if err != nil {
		return nil, fmt.Errorf("parse JWT private key reference: %w", err)
	}
	publicJWK, err := newPublicJWK(keyID, privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("build JWT public JWK: %w", err)
	}

	return &Signer{
		keyID:         keyID,
		privateKeyRef: privateKeyRef,
		privateKey:    privateKey,
		publicJWK:     publicJWK,
	}, nil
}

type KeyMaterial struct {
	KeyID         signingkey.KeyID
	Algorithm     signingkey.Algorithm
	PublicJWK     signingkey.PublicJWK
	PrivateKeyRef signingkey.PrivateKeyRef
}

func (s *Signer) KeyMaterial() KeyMaterial {
	return KeyMaterial{
		KeyID:         s.keyID,
		Algorithm:     signingkey.AlgorithmEdDSA,
		PublicJWK:     slices.Clone(s.publicJWK),
		PrivateKeyRef: s.privateKeyRef,
	}
}

func newPublicJWK(keyID signingkey.KeyID, publicKey ed25519.PublicKey) (signingkey.PublicJWK, error) {
	value, err := json.Marshal(map[string]string{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(publicKey),
		"kid": string(keyID),
		"alg": string(signingkey.AlgorithmEdDSA),
		"use": "sig",
	})
	if err != nil {
		return nil, err
	}
	return signingkey.NewPublicJWK(value)
}

func (s *Signer) Sign(ctx context.Context, input accesstokenservice.SignInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	if input.PrivateKeyRef != s.privateKeyRef {
		return "", errors.New("JWT signing key reference is not configured")
	}
	if input.Algorithm != signingkey.AlgorithmEdDSA {
		return "", errors.New("JWT signing algorithm is not supported")
	}
	if input.KeyID == "" {
		return "", errors.New("JWT signing key ID is required")
	}

	token := jwtlib.NewWithClaims(
		jwtlib.SigningMethodEdDSA,
		toTokenClaims(input.Claims),
	)

	token.Header["typ"] = accessTokenType
	token.Header["kid"] = string(input.KeyID)

	value, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return value, nil
}

func (s *Signer) SignIDToken(ctx context.Context, input idtokenservice.SignInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("sign ID token: %w", err)
	}
	if input.PrivateKeyRef != s.privateKeyRef {
		return "", errors.New("ID token signing key reference is not configured")
	}
	if input.Algorithm != signingkey.AlgorithmEdDSA {
		return "", errors.New("ID token signing algorithm is not supported")
	}
	if input.KeyID == "" {
		return "", errors.New("ID token signing key ID is required")
	}

	token := jwtlib.NewWithClaims(
		jwtlib.SigningMethodEdDSA,
		toIDTokenClaims(input.Claims),
	)
	token.Header["typ"] = idTokenType
	token.Header["kid"] = string(input.KeyID)

	value, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign ID token: %w", err)
	}

	return value, nil
}

type tokenClaims struct {
	ClientID  string `json:"client_id"`
	Scopes    string `json:"scope"`
	SessionID string `json:"sid"`

	AuthenticationLevel  string              `json:"acr"`
	AuthenticatedMethods []string            `json:"amr"`
	AuthenticatedAt      *jwtlib.NumericDate `json:"auth_time"`

	jwtlib.RegisteredClaims
}

type idTokenClaims struct {
	Nonce *string `json:"nonce,omitempty"`

	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`

	PhoneNumber         *string `json:"phone_number,omitempty"`
	PhoneNumberVerified *bool   `json:"phone_number_verified,omitempty"`

	Name       *string             `json:"name,omitempty"`
	FamilyName *string             `json:"family_name,omitempty"`
	GivenName  *string             `json:"given_name,omitempty"`
	MiddleName *string             `json:"middle_name,omitempty"`
	UpdatedAt  *jwtlib.NumericDate `json:"updated_at,omitempty"`

	AuthenticationLevel  string              `json:"acr"`
	AuthenticatedMethods []string            `json:"amr"`
	AuthenticatedAt      *jwtlib.NumericDate `json:"auth_time"`

	jwtlib.RegisteredClaims
}

func toTokenClaims(claims accesstokenservice.Claims) tokenClaims {
	return tokenClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			Audience:  jwtlib.ClaimStrings(slices.Clone(claims.Audience)),
			ExpiresAt: jwtlib.NewNumericDate(claims.ExpiresAt),
			IssuedAt:  jwtlib.NewNumericDate(claims.IssuedAt),
			ID:        claims.TokenID,
		},

		ClientID:  claims.ClientID,
		Scopes:    strings.Join(claims.Scopes, " "),
		SessionID: claims.SessionID,

		AuthenticationLevel:  claims.AuthenticationLevel,
		AuthenticatedMethods: slices.Clone(claims.AuthenticatedMethods),
		AuthenticatedAt:      jwtlib.NewNumericDate(claims.AuthenticatedAt),
	}
}

func toIDTokenClaims(claims idtokenservice.Claims) idTokenClaims {
	var updatedAt *jwtlib.NumericDate
	if claims.Identity.UpdatedAt != nil {
		updatedAt = jwtlib.NewNumericDate(*claims.Identity.UpdatedAt)
	}
	return idTokenClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			Audience:  jwtlib.ClaimStrings{claims.Audience},
			ExpiresAt: jwtlib.NewNumericDate(claims.ExpiresAt),
			IssuedAt:  jwtlib.NewNumericDate(claims.IssuedAt),
		},
		Nonce:                claims.Nonce,
		Email:                claims.Identity.Email,
		EmailVerified:        claims.Identity.EmailVerified,
		PhoneNumber:          claims.Identity.PhoneNumber,
		PhoneNumberVerified:  claims.Identity.PhoneNumberVerified,
		Name:                 claims.Identity.Name,
		FamilyName:           claims.Identity.FamilyName,
		GivenName:            claims.Identity.GivenName,
		MiddleName:           claims.Identity.MiddleName,
		UpdatedAt:            updatedAt,
		AuthenticationLevel:  claims.AuthenticationLevel,
		AuthenticatedMethods: slices.Clone(claims.AuthenticatedMethods),
		AuthenticatedAt:      jwtlib.NewNumericDate(claims.AuthenticatedAt),
	}
}

func loadPrivateKey(file string) (ed25519.PrivateKey, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read private key file: %w", err)
	}

	block, rest := pem.Decode(contents)
	if block == nil ||
		block.Type != "PRIVATE KEY" ||
		strings.TrimSpace(string(rest)) != "" {
		return nil, errors.New(
			"private key file must contain one PKCS#8 PEM block",
		)
	}

	value, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
	}

	privateKey, ok := value.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("private key must be Ed25519")
	}

	return slices.Clone(privateKey), nil
}
