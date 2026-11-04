package oauthclientadmin

import (
	"context"

	identityadminv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/admin/v1"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	oauthclientadminusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauthclientadmin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	CreatePublicClient(context.Context, oauthclientadminusecase.CreatePublicClientInput) (*oauthclientdomain.OAuthClient, error)
}

type Handler struct {
	identityadminv1.UnimplementedOAuthClientAdminServiceServer
	service Service
}

var _ identityadminv1.OAuthClientAdminServiceServer = (*Handler)(nil)

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreatePublicOAuthClient(
	ctx context.Context,
	request *identityadminv1.CreatePublicOAuthClientRequest,
) (*identityadminv1.CreatePublicOAuthClientResponse, error) {
	client, err := h.service.CreatePublicClient(ctx, oauthclientadminusecase.CreatePublicClientInput{
		ClientID:            request.GetClientId(),
		Name:                request.GetName(),
		Description:         request.Description,
		RedirectURIs:        request.GetRedirectUris(),
		Scopes:              request.GetScopes(),
		EnableRefreshTokens: request.GetEnableRefreshTokens(),
	})
	if err != nil {
		return nil, err
	}
	return &identityadminv1.CreatePublicOAuthClientResponse{OauthClient: toProto(client)}, nil
}

func toProto(client *oauthclientdomain.OAuthClient) *identityadminv1.OAuthClient {
	redirectURIs := make([]string, 0, len(client.RedirectURIs))
	for _, value := range client.RedirectURIs {
		redirectURIs = append(redirectURIs, string(value.URI))
	}
	scopes := make([]string, 0, len(client.Scopes))
	for _, value := range client.Scopes {
		scopes = append(scopes, string(value.Scope))
	}
	return &identityadminv1.OAuthClient{
		Id:                   client.ID.UUID().String(),
		ClientId:             string(client.ClientID),
		Name:                 client.Name,
		Description:          client.Description,
		RedirectUris:         redirectURIs,
		Scopes:               scopes,
		RefreshTokensEnabled: client.AllowsGrantType(oauthclientdomain.GrantTypeRefreshToken),
		RequirePkce:          client.RequirePKCE,
		CreatedAt:            timestamppb.New(client.CreatedAt),
		UpdatedAt:            timestamppb.New(client.UpdatedAt),
	}
}
