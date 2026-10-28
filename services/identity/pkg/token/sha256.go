package token

import (
	"crypto/sha256"
	"encoding/base64"
)

type SHA256 struct{}

func NewSHA256() *SHA256 {
	return &SHA256{}
}

func (*SHA256) Digest(token string) string {
	sum := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

var _ Digester = (*SHA256)(nil)
