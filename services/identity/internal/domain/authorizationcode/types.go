package authorizationcode

import (
	"errors"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Hash string

func NewHash(value string) (Hash, error) {
	if !domain.HasText(value) {
		return "", errors.New("authorization code hash is required")
	}

	return Hash(value), nil
}
