package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	metadatausecase "github.com/krackl1n/careeria/services/identity/internal/usecase/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type Service interface {
	GetOpenIDConfiguration() metadatausecase.OpenIDConfiguration
	GetJSONWebKeySet(context.Context) (metadatausecase.JSONWebKeySet, error)
}

type Handler struct {
	identityv1.UnimplementedIdentityMetadataServiceServer
	service Service
}

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetOpenIDConfiguration(context.Context, *identityv1.GetOpenIDConfigurationRequest) (*identityv1.GetOpenIDConfigurationResponse, error) {
	value := h.service.GetOpenIDConfiguration()
	return &identityv1.GetOpenIDConfigurationResponse{Issuer: value.Issuer, AuthorizationEndpoint: value.AuthorizationEndpoint, TokenEndpoint: value.TokenEndpoint, RevocationEndpoint: value.RevocationEndpoint, UserinfoEndpoint: value.UserInfoEndpoint, JwksUri: value.JSONWebKeySetURI, ResponseTypesSupported: value.ResponseTypesSupported, SubjectTypesSupported: value.SubjectTypesSupported, IdTokenSigningAlgValuesSupported: value.IDTokenSigningAlgorithmsSupported, ScopesSupported: value.ScopesSupported, ClaimsSupported: value.ClaimsSupported, GrantTypesSupported: value.GrantTypesSupported, TokenEndpointAuthMethodsSupported: value.TokenEndpointAuthMethodsSupported, RevocationEndpointAuthMethodsSupported: value.RevocationEndpointAuthMethodsSupported, CodeChallengeMethodsSupported: value.CodeChallengeMethodsSupported}, nil
}

func (h *Handler) GetJSONWebKeySet(ctx context.Context, _ *identityv1.GetJSONWebKeySetRequest) (*identityv1.GetJSONWebKeySetResponse, error) {
	value, err := h.service.GetJSONWebKeySet(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]*structpb.Struct, len(value.Keys))
	for i, key := range value.Keys {
		var payload map[string]any
		if err := json.Unmarshal(key, &payload); err != nil {
			return nil, fmt.Errorf("decode public JWK: %w", err)
		}
		keys[i], err = structpb.NewStruct(payload)
		if err != nil {
			return nil, fmt.Errorf("convert public JWK: %w", err)
		}
	}
	return &identityv1.GetJSONWebKeySetResponse{Keys: keys}, nil
}
