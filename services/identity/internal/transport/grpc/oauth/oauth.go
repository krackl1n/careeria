package oauth

import (
	"context"
	"fmt"
	"net/http"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) Authorize(
	ctx context.Context,
	request *identityv1.AuthorizeRequest,
) (*identityv1.AuthorizeResponse, error) {
	output, err := h.oauth.StartAuthorization(ctx, toStartAuthorizationInput(request))
	if err != nil {
		return nil, err
	}
	if err := grpchttpresponse.SetRedirect(ctx, http.StatusFound, output.RedirectURI); err != nil {
		return nil, fmt.Errorf("set authorization redirect metadata: %w", err)
	}

	return toAuthorizeResponse(output), nil
}

func (h *Handler) ExchangeToken(
	ctx context.Context,
	request *identityv1.ExchangeTokenRequest,
) (*identityv1.ExchangeTokenResponse, error) {
	switch request.GetGrantType() {
	case "authorization_code":
		input, err := toExchangeAuthorizationCodeInput(ctx, request)
		if err != nil {
			return nil, err
		}
		output, err := h.oauth.ExchangeAuthorizationCode(ctx, input)
		if err != nil {
			return nil, err
		}
		return toExchangeAuthorizationCodeResponse(output), nil
	case "refresh_token":
		input, err := toRefreshAccessTokenInput(ctx, request)
		if err != nil {
			return nil, err
		}
		output, err := h.oauth.RefreshAccessToken(ctx, input)
		if err != nil {
			return nil, err
		}
		return toRefreshAccessTokenResponse(output), nil
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported OAuth grant type")
	}
}

func (h *Handler) ContinueAuthorization(
	ctx context.Context,
	request *identityv1.ContinueAuthorizationRequest,
) (*identityv1.ContinueAuthorizationResponse, error) {
	output, err := h.oauth.ContinueAuthorization(ctx, toContinueAuthorizationInput(request))
	if err != nil {
		return nil, err
	}
	if err := grpchttpresponse.SetRedirect(ctx, http.StatusFound, output.RedirectURI); err != nil {
		return nil, fmt.Errorf("set continued authorization redirect metadata: %w", err)
	}

	return &identityv1.ContinueAuthorizationResponse{RedirectUri: output.RedirectURI}, nil
}
