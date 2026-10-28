package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (s *Service) getAndAuthenticateClient(
	ctx context.Context,
	clientID oauthclientdomain.ClientID,
	clientSecret *string,
	now time.Time,
) (*oauthclientdomain.OAuthClient, error) {
	client, err := s.oauthClients.GetByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, oauthclientdomain.ErrNotFound) {
			return nil, invalidClient(err)
		}
		return nil, fmt.Errorf("get OAuth client for token endpoint: %w", err)
	}

	switch client.TokenEndpointAuthMethod {
	case oauthclientdomain.TokenEndpointAuthMethodNone:
		if clientSecret != nil {
			return nil, invalidClient(errors.New("public OAuth client must not use client secret authentication"))
		}
	case oauthclientdomain.TokenEndpointAuthMethodClientSecretBasic:
		if clientSecret == nil || *clientSecret == "" {
			return nil, invalidClient(errors.New("OAuth client secret is required"))
		}
		var credentialID *oauthclientdomain.CredentialID
		for index := range client.Credentials {
			credential := &client.Credentials[index]
			if !credential.IsUsable(now) {
				continue
			}
			matches, err := s.clientSecrets.Verify(*clientSecret, string(credential.SecretHash))
			if err != nil {
				return nil, fmt.Errorf("verify OAuth client secret: %w", err)
			}
			if matches {
				credentialID = &credential.ID
			}
		}
		if credentialID == nil {
			return nil, invalidClient(errors.New("OAuth client credentials are invalid"))
		}
		if err := client.RecordCredentialUsed(*credentialID, now); err != nil {
			return nil, invalidClient(err)
		}
		if err := s.oauthClients.Update(ctx, client); err != nil {
			return nil, fmt.Errorf("record OAuth client credential use: %w", err)
		}
	default:
		return nil, invalidClient(errors.New("OAuth client authentication method is unsupported"))
	}

	return client, nil
}
