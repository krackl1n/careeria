package oauth

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestAuthorizeSetsRedirectMetadata(t *testing.T) {
	transactionID := uuid.New()
	expiresAt := time.Now().Add(time.Minute).UTC()
	service := &serviceStub{
		startAuthorization: func(_ context.Context, input oauthusecase.StartAuthorizationInput) (oauthusecase.StartAuthorizationOutput, error) {
			require.Equal(t, "code", input.ResponseType)
			require.Equal(t, "web", input.ClientID)
			return oauthusecase.StartAuthorizationOutput{
				RedirectURI:                "https://identity.example/login",
				AuthorizationTransactionID: transactionID,
				ExpiresAt:                  expiresAt,
			}, nil
		},
	}
	stream := &serverTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)

	response, err := NewHandler(service).Authorize(ctx, &identityv1.AuthorizeRequest{
		ResponseType: "code",
		ClientId:     "web",
	})

	require.NoError(t, err)
	require.Equal(t, "https://identity.example/login", response.GetRedirectUri())
	require.Equal(t, transactionID.String(), response.GetAuthorizationTransactionId())
	require.Equal(t, expiresAt, response.GetExpiresAt().AsTime())
	requireRedirectMetadata(t, stream.headers, "https://identity.example/login")
}

func TestContinueAuthorizationSetsRedirectMetadata(t *testing.T) {
	transactionID := uuid.New()
	service := &serviceStub{
		continueAuthorization: func(_ context.Context, input oauthusecase.ContinueAuthorizationInput) (oauthusecase.ContinueAuthorizationOutput, error) {
			require.Equal(t, transactionID, input.AuthorizationTransactionID)
			return oauthusecase.ContinueAuthorizationOutput{
				RedirectURI: "https://client.example/callback?code=secret",
			}, nil
		},
	}
	stream := &serverTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)

	response, err := NewHandler(service).ContinueAuthorization(ctx, &identityv1.ContinueAuthorizationRequest{
		AuthorizationTransactionId: transactionID.String(),
	})

	require.NoError(t, err)
	require.Equal(t, "https://client.example/callback?code=secret", response.GetRedirectUri())
	requireRedirectMetadata(t, stream.headers, "https://client.example/callback?code=secret")
}

func TestExchangeTokenPassesBasicClientAuthentication(t *testing.T) {
	clientID := "client-id"
	redirectURI := "https://client.example/callback"
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	refreshToken := "refresh-token"
	service := &serviceStub{
		exchangeAuthorizationCode: func(_ context.Context, input oauthusecase.ExchangeAuthorizationCodeInput) (oauthusecase.ExchangeAuthorizationCodeOutput, error) {
			require.Equal(t, clientID, input.ClientID)
			require.Equal(t, "secret:value", *input.ClientSecret)
			require.Equal(t, "code-value", input.Code)
			require.Equal(t, redirectURI, input.RedirectURI)
			require.Equal(t, &verifier, input.CodeVerifier)
			return oauthusecase.ExchangeAuthorizationCodeOutput{
				AccessToken:  "access-token",
				TokenType:    "Bearer",
				ExpiresIn:    900,
				RefreshToken: &refreshToken,
				IDToken:      "id-token",
				Scope:        "openid profile",
			}, nil
		},
	}
	stream := &serverTransportStream{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte("client-id:secret%3Avalue")),
	))
	ctx = grpc.NewContextWithServerTransportStream(ctx, stream)

	response, err := NewHandler(service).ExchangeToken(ctx, &identityv1.ExchangeTokenRequest{
		GrantType: "authorization_code",
		ClientId:  &clientID,
		GrantCredential: &identityv1.ExchangeTokenRequest_Code{
			Code: "code-value",
		},
		RedirectUri:  &redirectURI,
		CodeVerifier: &verifier,
	})

	require.NoError(t, err)
	require.Equal(t, "access-token", response.GetAccessToken())
	require.Equal(t, "Bearer", response.GetTokenType())
	require.Equal(t, uint32(900), response.GetExpiresIn())
	require.Equal(t, refreshToken, response.GetRefreshToken())
	require.Equal(t, "openid profile", response.GetScope())
}

func TestExchangeTokenPreservesOAuthErrorCode(t *testing.T) {
	service := &serviceStub{
		refreshAccessToken: func(context.Context, oauthusecase.RefreshAccessTokenInput) (oauthusecase.RefreshAccessTokenOutput, error) {
			return oauthusecase.RefreshAccessTokenOutput{}, apperr.New(
				apperr.TypeInvalidArgument,
				"oauth_invalid_grant",
				"OAuth grant is invalid",
				nil,
			)
		},
	}
	clientID := "web"

	_, err := NewHandler(service).ExchangeToken(context.Background(), &identityv1.ExchangeTokenRequest{
		GrantType: "refresh_token",
		ClientId:  &clientID,
		GrantCredential: &identityv1.ExchangeTokenRequest_RefreshToken{
			RefreshToken: "refresh-token",
		},
	})

	require.Error(t, err)
	value, ok := apperr.As(err)
	require.True(t, ok)
	require.Equal(t, "oauth_invalid_grant", value.Code())
	require.Equal(t, "OAuth grant is invalid", value.Message())
}

func requireRedirectMetadata(t *testing.T, md metadata.MD, location string) {
	t.Helper()
	require.Equal(t, []string{"302"}, md.Get(grpchttpresponse.StatusCodeMetadataKey))
	require.Equal(t, []string{location}, md.Get(grpchttpresponse.LocationMetadataKey))
}

type serviceStub struct {
	startAuthorization        func(context.Context, oauthusecase.StartAuthorizationInput) (oauthusecase.StartAuthorizationOutput, error)
	continueAuthorization     func(context.Context, oauthusecase.ContinueAuthorizationInput) (oauthusecase.ContinueAuthorizationOutput, error)
	exchangeAuthorizationCode func(context.Context, oauthusecase.ExchangeAuthorizationCodeInput) (oauthusecase.ExchangeAuthorizationCodeOutput, error)
	refreshAccessToken        func(context.Context, oauthusecase.RefreshAccessTokenInput) (oauthusecase.RefreshAccessTokenOutput, error)
}

func (s *serviceStub) StartAuthorization(ctx context.Context, input oauthusecase.StartAuthorizationInput) (oauthusecase.StartAuthorizationOutput, error) {
	return s.startAuthorization(ctx, input)
}

func (s *serviceStub) ContinueAuthorization(ctx context.Context, input oauthusecase.ContinueAuthorizationInput) (oauthusecase.ContinueAuthorizationOutput, error) {
	return s.continueAuthorization(ctx, input)
}

func (s *serviceStub) ExchangeAuthorizationCode(ctx context.Context, input oauthusecase.ExchangeAuthorizationCodeInput) (oauthusecase.ExchangeAuthorizationCodeOutput, error) {
	return s.exchangeAuthorizationCode(ctx, input)
}

func (s *serviceStub) RefreshAccessToken(ctx context.Context, input oauthusecase.RefreshAccessTokenInput) (oauthusecase.RefreshAccessTokenOutput, error) {
	return s.refreshAccessToken(ctx, input)
}

type serverTransportStream struct {
	headers metadata.MD
}

func (*serverTransportStream) Method() string {
	return "/careeria.identity.v1.OAuthService/Authorize"
}

func (s *serverTransportStream) SetHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (s *serverTransportStream) SendHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (*serverTransportStream) SetTrailer(metadata.MD) error {
	return nil
}
