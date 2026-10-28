package metadata

import (
	"context"
	"errors"
	"fmt"
	"slices"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type JSONWebKeySet struct {
	Keys []signingkeydomain.PublicJWK
}

func (s *Service) GetJSONWebKeySet(ctx context.Context) (JSONWebKeySet, error) {
	now := s.clock.Now()
	keys, err := s.keys.ListPublic(ctx, now)
	if err != nil {
		return JSONWebKeySet{}, fmt.Errorf("get JSON Web Key Set: list public signing keys: %w", err)
	}
	if len(keys) == 0 {
		return JSONWebKeySet{}, errors.New("get JSON Web Key Set: no public signing keys")
	}

	result := JSONWebKeySet{Keys: make([]signingkeydomain.PublicJWK, len(keys))}
	for index, key := range keys {
		if err := key.ValidateForVerification(now); err != nil {
			return JSONWebKeySet{}, fmt.Errorf("get JSON Web Key Set: signing key cannot be published: %w", err)
		}
		result.Keys[index] = slices.Clone(key.PublicJWK)
	}

	return result, nil
}
