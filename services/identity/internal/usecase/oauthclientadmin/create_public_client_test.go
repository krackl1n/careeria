package oauthclientadmin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestCreatePublicClientCreatesPKCEClient(t *testing.T) {
	repository := &clientRepository{}
	service := NewService(repository, transactionManager{}, uuidGenerator{}, fixedClock{})

	client, err := service.CreatePublicClient(context.Background(), validInput())

	require.NoError(t, err)
	require.Equal(t, oauthclientdomain.ClientTypePublic, client.ClientType)
	require.Equal(t, oauthclientdomain.TokenEndpointAuthMethodNone, client.TokenEndpointAuthMethod)
	require.True(t, client.RequirePKCE)
	require.True(t, client.AllowsGrantType(oauthclientdomain.GrantTypeAuthorizationCode))
	require.True(t, client.AllowsGrantType(oauthclientdomain.GrantTypeRefreshToken))
	require.Len(t, client.RedirectURIs, 1)
	require.Len(t, client.Scopes, 4)
	require.Equal(t, 1, repository.createCalls)
}

func TestCreatePublicClientIsIdempotentForSameConfiguration(t *testing.T) {
	repository := &clientRepository{}
	service := NewService(repository, transactionManager{}, uuidGenerator{}, fixedClock{})

	first, err := service.CreatePublicClient(context.Background(), validInput())
	require.NoError(t, err)
	second, err := service.CreatePublicClient(context.Background(), validInput())

	require.NoError(t, err)
	require.Same(t, first, second)
	require.Equal(t, 1, repository.createCalls)
}

func TestCreatePublicClientRejectsDifferentConfigurationForExistingClientID(t *testing.T) {
	repository := &clientRepository{}
	service := NewService(repository, transactionManager{}, uuidGenerator{}, fixedClock{})

	_, err := service.CreatePublicClient(context.Background(), validInput())
	require.NoError(t, err)
	input := validInput()
	input.Name = "Another application"
	_, err = service.CreatePublicClient(context.Background(), input)

	require.Error(t, err)
	require.True(t, apperr.IsType(err, apperr.TypeConflict))
	require.Equal(t, 1, repository.createCalls)
}

func TestCreatePublicClientRequiresOpenIDScope(t *testing.T) {
	repository := &clientRepository{}
	service := NewService(repository, transactionManager{}, uuidGenerator{}, fixedClock{})
	input := validInput()
	input.Scopes = []string{"profile"}

	_, err := service.CreatePublicClient(context.Background(), input)

	require.Error(t, err)
	require.True(t, apperr.IsType(err, apperr.TypeInvalidArgument))
	require.Zero(t, repository.createCalls)
}

func validInput() CreatePublicClientInput {
	return CreatePublicClientInput{
		ClientID:            "careeria-local",
		Name:                "RVSpace Web",
		RedirectURIs:        []string{"http://localhost:13000/auth/callback"},
		Scopes:              []string{"openid", "profile", "email", "phone"},
		EnableRefreshTokens: true,
	}
}

type clientRepository struct {
	client      *oauthclientdomain.OAuthClient
	createCalls int
}

func (r *clientRepository) Create(_ context.Context, client *oauthclientdomain.OAuthClient) error {
	if r.client != nil {
		return oauthclientdomain.ErrClientIDAlreadyExists
	}
	r.client = client
	r.createCalls++
	return nil
}

func (r *clientRepository) GetByClientID(_ context.Context, clientID oauthclientdomain.ClientID) (*oauthclientdomain.OAuthClient, error) {
	if r.client == nil || r.client.ClientID != clientID {
		return nil, oauthclientdomain.ErrNotFound
	}
	return r.client, nil
}

type transactionManager struct{}

func (transactionManager) Do(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type uuidGenerator struct{}

func (uuidGenerator) New() uuid.UUID { return uuid.New() }

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
}
