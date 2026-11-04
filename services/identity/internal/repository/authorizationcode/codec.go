package authorizationcode

import (
	"encoding/json"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
)

const authorizationCodeKeyPrefix = "identity:oauth:authorization-codes:"

func key(hash domain.Hash) string {
	return authorizationCodeKeyPrefix + string(hash)
}

func encode(entity *domain.AuthorizationCode) ([]byte, error) {
	payload, err := json.Marshal(toModel(entity))
	if err != nil {
		return nil, fmt.Errorf("encode authorization code: %w", err)
	}

	return payload, nil
}

func decode(payload []byte) (*domain.AuthorizationCode, error) {
	var model model
	if err := json.Unmarshal(payload, &model); err != nil {
		return nil, fmt.Errorf("decode authorization code: %w", err)
	}

	entity, err := model.toDomain()
	if err != nil {
		return nil, fmt.Errorf("decode authorization code: %w", err)
	}

	return entity, nil
}
