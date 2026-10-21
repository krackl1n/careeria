package signingkeyadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (s *Service) CreateSigningKey(ctx context.Context) (*signingkeydomain.SigningKey, error) {
	key, err := s.keys.GetByKeyID(ctx, s.material.KeyID)
	switch {
	case err == nil:
		return s.activateExisting(ctx, key)
	case !errors.Is(err, signingkeydomain.ErrNotFound):
		return nil, fmt.Errorf("get signing key by key ID: %w", err)
	}

	now := s.clock.Now()
	key, err = signingkeydomain.New(
		signingkeydomain.ID(s.ids.New()),
		s.material.KeyID,
		s.material.Algorithm,
		s.material.PublicJWK,
		s.material.PrivateKeyRef,
		now,
		nil,
		now,
	)
	if err != nil {
		return nil, invalidSigningKey(err)
	}
	if err := key.Activate(now); err != nil {
		return nil, invalidSigningKey(err)
	}
	if err := s.keys.Create(ctx, key); err != nil {
		switch {
		case errors.Is(err, signingkeydomain.ErrKeyIDAlreadyExists):
			existing, getErr := s.keys.GetByKeyID(ctx, s.material.KeyID)
			if getErr == nil {
				return s.activateExisting(ctx, existing)
			}
			return nil, fmt.Errorf("get concurrently created signing key: %w", getErr)
		case errors.Is(err, signingkeydomain.ErrActiveAlgorithmConflict):
			return nil, signingKeyConflict(err)
		default:
			return nil, fmt.Errorf("create signing key: %w", err)
		}
	}
	return key, nil
}

func (s *Service) activateExisting(
	ctx context.Context,
	key *signingkeydomain.SigningKey,
) (*signingkeydomain.SigningKey, error) {
	if key == nil ||
		key.Algorithm != s.material.Algorithm ||
		key.PrivateKeyRef != s.material.PrivateKeyRef ||
		!equalJSON(key.PublicJWK, s.material.PublicJWK) {
		return nil, signingKeyConflict(errors.New("signing key metadata does not match configured key"))
	}

	switch key.Status {
	case signingkeydomain.StatusPending:
		if err := key.Activate(s.clock.Now()); err != nil {
			return nil, invalidSigningKey(err)
		}
		if err := s.keys.Update(ctx, key); err != nil {
			if errors.Is(err, signingkeydomain.ErrActiveAlgorithmConflict) {
				return nil, signingKeyConflict(err)
			}
			return nil, fmt.Errorf("activate signing key: %w", err)
		}
		return key, nil
	case signingkeydomain.StatusActive:
		if err := key.ValidateForSigning(s.clock.Now()); err != nil {
			return nil, invalidSigningKey(err)
		}
		return key, nil
	default:
		return nil, signingKeyConflict(fmt.Errorf("signing key has status %s", key.Status))
	}
}

func equalJSON(left, right []byte) bool {
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func invalidSigningKey(err error) error {
	return apperr.New(
		apperr.TypeInvalidArgument,
		"signing_key_create_invalid",
		"Signing key is invalid",
		err,
	)
}

func signingKeyConflict(err error) error {
	return apperr.New(
		apperr.TypeConflict,
		"signing_key_conflict",
		"Signing key conflicts with existing key",
		err,
	)
}
