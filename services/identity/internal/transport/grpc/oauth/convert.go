package oauth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toStartAuthorizationInput(request *identityv1.AuthorizeRequest) oauthusecase.StartAuthorizationInput {
	return oauthusecase.StartAuthorizationInput{
		ResponseType:        request.GetResponseType(),
		ClientID:            request.GetClientId(),
		RedirectURI:         request.GetRedirectUri(),
		Scope:               request.GetScope(),
		State:               request.State,
		Nonce:               request.Nonce,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
	}
}

func toExchangeAuthorizationCodeInput(
	ctx context.Context,
	request *identityv1.ExchangeTokenRequest,
) (oauthusecase.ExchangeAuthorizationCodeInput, error) {
	clientID, clientSecret, err := tokenClientCredentials(ctx, request)
	if err != nil {
		return oauthusecase.ExchangeAuthorizationCodeInput{}, err
	}

	return oauthusecase.ExchangeAuthorizationCodeInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Code:         request.GetCode(),
		RedirectURI:  request.GetRedirectUri(),
		CodeVerifier: request.CodeVerifier,
	}, nil
}

func toRefreshAccessTokenInput(
	ctx context.Context,
	request *identityv1.ExchangeTokenRequest,
) (oauthusecase.RefreshAccessTokenInput, error) {
	clientID, clientSecret, err := tokenClientCredentials(ctx, request)
	if err != nil {
		return oauthusecase.RefreshAccessTokenInput{}, err
	}

	return oauthusecase.RefreshAccessTokenInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: request.GetRefreshToken(),
	}, nil
}

func tokenClientCredentials(
	ctx context.Context,
	request *identityv1.ExchangeTokenRequest,
) (string, *string, error) {
	authentication, err := clientAuthenticationFromContext(ctx)
	if err != nil {
		return "", nil, status.Error(codes.Unauthenticated, "OAuth client authentication failed")
	}
	if authentication == nil {
		return request.GetClientId(), nil, nil
	}
	if request.ClientId != nil && request.GetClientId() != authentication.clientID {
		return "", nil, status.Error(codes.Unauthenticated, "OAuth client authentication failed")
	}

	return authentication.clientID, &authentication.clientSecret, nil
}

type clientAuthentication struct {
	clientID     string
	clientSecret string
}

func clientAuthenticationFromContext(ctx context.Context) (*clientAuthentication, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, errors.New("authorization header must contain one value")
	}

	scheme, encoded, ok := strings.Cut(values[0], " ")
	if !ok || !strings.EqualFold(scheme, "Basic") || encoded == "" {
		return nil, errors.New("client authentication scheme must be Basic")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("client authentication is malformed")
	}
	clientID, clientSecret, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return nil, errors.New("client authentication is malformed")
	}
	clientID, err = url.QueryUnescape(clientID)
	if err != nil {
		return nil, errors.New("client identifier is malformed")
	}
	clientSecret, err = url.QueryUnescape(clientSecret)
	if err != nil {
		return nil, errors.New("client secret is malformed")
	}

	return &clientAuthentication{
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

func toExchangeAuthorizationCodeResponse(
	output oauthusecase.ExchangeAuthorizationCodeOutput,
) *identityv1.ExchangeTokenResponse {
	return &identityv1.ExchangeTokenResponse{
		AccessToken:  output.AccessToken,
		TokenType:    output.TokenType,
		ExpiresIn:    output.ExpiresIn,
		RefreshToken: output.RefreshToken,
		IdToken:      &output.IDToken,
		Scope:        output.Scope,
	}
}

func toRefreshAccessTokenResponse(
	output oauthusecase.RefreshAccessTokenOutput,
) *identityv1.ExchangeTokenResponse {
	return &identityv1.ExchangeTokenResponse{
		AccessToken:  output.AccessToken,
		TokenType:    output.TokenType,
		ExpiresIn:    output.ExpiresIn,
		RefreshToken: &output.RefreshToken,
		Scope:        output.Scope,
	}
}

func toContinueAuthorizationInput(request *identityv1.ContinueAuthorizationRequest) oauthusecase.ContinueAuthorizationInput {
	id, _ := uuid.Parse(request.GetAuthorizationTransactionId())
	return oauthusecase.ContinueAuthorizationInput{AuthorizationTransactionID: id}
}

func toAuthorizeResponse(output oauthusecase.StartAuthorizationOutput) *identityv1.AuthorizeResponse {
	return &identityv1.AuthorizeResponse{
		RedirectUri:                output.RedirectURI,
		AuthorizationTransactionId: output.AuthorizationTransactionID.String(),
		ExpiresAt:                  timestamppb.New(output.ExpiresAt),
	}
}
