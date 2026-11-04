package openidconnect

import "errors"

type Nonce string

const maxNonceLength = 2048

func NewNonce(value string) (Nonce, error) {
	if value == "" {
		return "", errors.New("OpenID Connect nonce is required")
	}
	if len(value) > maxNonceLength {
		return "", errors.New("OpenID Connect nonce exceeds 2048 bytes")
	}

	return Nonce(value), nil
}
