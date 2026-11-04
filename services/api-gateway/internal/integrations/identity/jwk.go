package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
)

func (c *Clients) LoadJWKS(ctx context.Context) (map[string][]byte, error) {
	if c == nil || c.Metadata == nil {
		return nil, errors.New("identity metadata client is required")
	}
	response, err := c.Metadata.GetJSONWebKeySet(ctx, &identityv1.GetJSONWebKeySetRequest{})
	if err != nil {
		return nil, fmt.Errorf("get JSON Web Key Set: %w", err)
	}

	jwks := make(map[string][]byte, len(response.GetKeys()))
	for _, key := range response.GetKeys() {
		if key == nil {
			return nil, errors.New("JSON Web Key Set contains an empty key")
		}
		kid := key.GetFields()["kid"].GetStringValue()
		if kid == "" {
			return nil, errors.New("JSON Web Key Set contains a key without kid")
		}
		if _, exists := jwks[kid]; exists {
			return nil, fmt.Errorf("JSON Web Key Set contains duplicate key %q", kid)
		}
		value, err := json.Marshal(key.AsMap())
		if err != nil {
			return nil, fmt.Errorf("encode public key %q: %w", kid, err)
		}
		jwks[kid] = value
	}
	if len(jwks) == 0 {
		return nil, errors.New("JSON Web Key Set is empty")
	}
	return jwks, nil
}
