package oauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Scope string

func NewScope(value string) (Scope, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("OAuth scope must be a non-empty token")
	}

	return Scope(value), nil
}

type CodeChallenge string

func NewCodeChallenge(value string) (CodeChallenge, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return "", errors.New("PKCE code challenge must be a canonical base64url-encoded SHA-256 value")
	}

	return CodeChallenge(value), nil
}

type CodeChallengeMethod string

const CodeChallengeMethodS256 CodeChallengeMethod = "S256"

func ParseCodeChallengeMethod(value string) (CodeChallengeMethod, error) {
	method := CodeChallengeMethod(value)
	if method != CodeChallengeMethodS256 {
		return "", errors.New("unsupported PKCE code challenge method")
	}

	return method, nil
}

type CodeVerifier string

func NewCodeVerifier(value string) (CodeVerifier, error) {
	if len(value) < 43 || len(value) > 128 {
		return "", errors.New("PKCE code verifier length must be between 43 and 128 characters")
	}
	for _, character := range value {
		if !isPKCEUnreserved(character) {
			return "", errors.New("PKCE code verifier contains an invalid character")
		}
	}

	return CodeVerifier(value), nil
}

type PKCE struct {
	Challenge CodeChallenge
	Method    CodeChallengeMethod
}

func NewPKCE(rawChallenge, rawMethod *string) (*PKCE, error) {
	switch {
	case rawChallenge == nil && rawMethod == nil:
		return nil, nil
	case rawChallenge == nil:
		return nil, errors.New("PKCE code challenge is required when method is specified")
	case rawMethod == nil:
		return nil, errors.New("PKCE code challenge method S256 is required")
	}

	challenge, err := NewCodeChallenge(*rawChallenge)
	if err != nil {
		return nil, err
	}
	method, err := ParseCodeChallengeMethod(*rawMethod)
	if err != nil {
		return nil, err
	}

	return &PKCE{Challenge: challenge, Method: method}, nil
}

func (p PKCE) Verify(verifier CodeVerifier) bool {
	digest := sha256.Sum256([]byte(verifier))
	encoded := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(encoded), []byte(p.Challenge)) == 1
}

func isPKCEUnreserved(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		character == '-' || character == '.' || character == '_' || character == '~'
}
