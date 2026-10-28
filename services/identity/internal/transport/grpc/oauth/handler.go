package oauth

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
)

type Handler struct {
	identityv1.UnimplementedOAuthServiceServer

	oauth Service
}

type Service interface {
	StartAuthorization(context.Context, oauthusecase.StartAuthorizationInput) (oauthusecase.StartAuthorizationOutput, error)
	ContinueAuthorization(context.Context, oauthusecase.ContinueAuthorizationInput) (oauthusecase.ContinueAuthorizationOutput, error)
	ExchangeAuthorizationCode(context.Context, oauthusecase.ExchangeAuthorizationCodeInput) (oauthusecase.ExchangeAuthorizationCodeOutput, error)
	RefreshAccessToken(context.Context, oauthusecase.RefreshAccessTokenInput) (oauthusecase.RefreshAccessTokenOutput, error)
}

var _ identityv1.OAuthServiceServer = (*Handler)(nil)

func NewHandler(oauth Service) *Handler {
	return &Handler{oauth: oauth}
}
