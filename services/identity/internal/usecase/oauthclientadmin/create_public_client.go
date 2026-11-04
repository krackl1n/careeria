package oauthclientadmin

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type CreatePublicClientInput struct {
	ClientID            string
	Name                string
	Description         *string
	RedirectURIs        []string
	Scopes              []string
	EnableRefreshTokens bool
}

func (s *Service) CreatePublicClient(
	ctx context.Context,
	input CreatePublicClientInput,
) (*oauthclientdomain.OAuthClient, error) {
	client, err := s.newPublicClient(input)
	if err != nil {
		return nil, invalidClient(err)
	}

	existing, err := s.clients.GetByClientID(ctx, client.ClientID)
	switch {
	case err == nil:
		if sameClient(existing, client) {
			return existing, nil
		}
		return nil, clientIDConflict(oauthclientdomain.ErrClientIDAlreadyExists)
	case !errors.Is(err, oauthclientdomain.ErrNotFound):
		return nil, fmt.Errorf("get OAuth client by client ID: %w", err)
	}

	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		if err := s.clients.Create(txCtx, client); err != nil {
			if errors.Is(err, oauthclientdomain.ErrClientIDAlreadyExists) {
				return clientIDConflict(err)
			}
			return fmt.Errorf("create OAuth client: %w", err)
		}
		return nil
	})
	if err != nil {
		if apperr.IsType(err, apperr.TypeConflict) {
			existing, getErr := s.clients.GetByClientID(ctx, client.ClientID)
			if getErr == nil && sameClient(existing, client) {
				return existing, nil
			}
		}
		return nil, err
	}
	return client, nil
}

func (s *Service) newPublicClient(input CreatePublicClientInput) (*oauthclientdomain.OAuthClient, error) {
	if len(input.RedirectURIs) == 0 {
		return nil, errors.New("redirect URI is required")
	}
	clientID, err := oauthclientdomain.NewClientID(input.ClientID)
	if err != nil {
		return nil, err
	}
	initialRedirectURI, err := oauthclientdomain.NewRedirectURI(input.RedirectURIs[0])
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	client, err := oauthclientdomain.NewPublic(
		oauthclientdomain.ID(s.ids.New()),
		clientID,
		input.Name,
		input.Description,
		nil,
		oauthclientdomain.RedirectURIID(s.ids.New()),
		initialRedirectURI,
		now,
	)
	if err != nil {
		return nil, err
	}
	for _, rawURI := range input.RedirectURIs[1:] {
		uri, err := oauthclientdomain.NewRedirectURI(rawURI)
		if err != nil {
			return nil, err
		}
		if err := client.AddRedirectURI(oauthclientdomain.RedirectURIID(s.ids.New()), uri, now); err != nil {
			return nil, err
		}
	}

	hasOpenID := false
	for _, rawScope := range input.Scopes {
		scope, err := oauthdomain.NewScope(rawScope)
		if err != nil {
			return nil, err
		}
		if scope == "openid" {
			hasOpenID = true
		}
		if err := client.AddScope(scope, now); err != nil {
			return nil, err
		}
	}
	if !hasOpenID {
		return nil, errors.New("openid scope is required")
	}
	if input.EnableRefreshTokens {
		if err := client.AddGrantType(oauthclientdomain.GrantTypeRefreshToken, now); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func sameClient(existing, requested *oauthclientdomain.OAuthClient) bool {
	if existing == nil ||
		existing.ClientType != oauthclientdomain.ClientTypePublic ||
		existing.Status != oauthclientdomain.StatusActive ||
		existing.ClientID != requested.ClientID ||
		existing.Name != requested.Name ||
		existing.RequirePKCE != requested.RequirePKCE ||
		existing.TokenEndpointAuthMethod != requested.TokenEndpointAuthMethod ||
		existing.ConsentMode != requested.ConsentMode ||
		!equalOptionalString(existing.Description, requested.Description) {
		return false
	}
	return slices.Equal(sortedRedirectURIs(existing), sortedRedirectURIs(requested)) &&
		slices.Equal(sortedScopes(existing), sortedScopes(requested)) &&
		slices.Equal(sortedGrantTypes(existing), sortedGrantTypes(requested))
}

func sortedRedirectURIs(client *oauthclientdomain.OAuthClient) []string {
	values := make([]string, 0, len(client.RedirectURIs))
	for _, value := range client.RedirectURIs {
		values = append(values, string(value.URI))
	}
	slices.Sort(values)
	return values
}

func sortedScopes(client *oauthclientdomain.OAuthClient) []string {
	values := make([]string, 0, len(client.Scopes))
	for _, value := range client.Scopes {
		values = append(values, string(value.Scope))
	}
	slices.Sort(values)
	return values
}

func sortedGrantTypes(client *oauthclientdomain.OAuthClient) []string {
	values := make([]string, 0, len(client.GrantTypes))
	for _, value := range client.GrantTypes {
		values = append(values, string(value.GrantType))
	}
	slices.Sort(values)
	return values
}

func equalOptionalString(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func invalidClient(err error) error {
	return apperr.New(
		apperr.TypeInvalidArgument,
		"oauth_client_create_invalid",
		"OAuth client is invalid",
		err,
	)
}

func clientIDConflict(err error) error {
	return apperr.New(
		apperr.TypeConflict,
		"oauth_client_id_already_exists",
		"OAuth client ID already exists",
		err,
	)
}
