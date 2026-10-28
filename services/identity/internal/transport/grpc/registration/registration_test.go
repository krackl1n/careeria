package registration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
	"github.com/stretchr/testify/require"
)

func TestRegisterIdentityUsesReturnedTransaction(t *testing.T) {
	transactionID := uuid.New()
	transaction := registrationContext(t, transactionID).AuthorizationTransaction
	service := &serviceStub{
		registerIdentity: func(_ context.Context, input authenticationusecase.RegisterIdentityInput) (authenticationusecase.RegisterIdentityOutput, error) {
			require.Equal(t, transactionID, input.AuthorizationTransactionID)
			require.Equal(t, "new-user@example.com", *input.Email)
			require.Equal(t, "secret123", input.Password)
			require.Equal(t, "Иванов", input.Profile.FamilyName)
			return authenticationusecase.RegisterIdentityOutput{AuthorizationTransaction: transaction}, nil
		},
		loadContext: func(_ context.Context, actual *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error) {
			require.Same(t, transaction, actual)
			return registrationContext(t, transactionID), nil
		},
	}

	response, err := NewHandler(service).RegisterIdentity(t.Context(), &identityv1.RegisterIdentityRequest{
		AuthorizationTransactionId: transactionID.String(),
		Contact: &identityv1.RegisterIdentityRequest_Email{
			Email: "new-user@example.com",
		},
		Password: "secret123",
		Profile: &identityv1.RegistrationProfile{
			FamilyName: "Иванов",
			GivenName:  "Иван",
		},
	})

	require.NoError(t, err)
	require.NoError(t, response.ValidateAll())
}

func registrationContext(t *testing.T, transactionID uuid.UUID) authenticationusecase.ContextOutput {
	t.Helper()
	clientID, err := oauthclientdomain.NewClientID("web")
	require.NoError(t, err)
	return authenticationusecase.ContextOutput{
		AuthorizationTransaction: &transactiondomain.AuthorizationTransaction{
			ID:              transactiondomain.ID(transactionID),
			RequestedScopes: []oauthdomain.Scope{"openid"},
			NextStep:        transactiondomain.NextStepChooseAuthenticationMethod,
			ExpiresAt:       time.Now().Add(time.Minute).UTC(),
		},
		OAuthClient: &oauthclientdomain.OAuthClient{ClientID: clientID, Name: "RVSpace Web"},
	}
}

type serviceStub struct {
	registerIdentity func(context.Context, authenticationusecase.RegisterIdentityInput) (authenticationusecase.RegisterIdentityOutput, error)
	loadContext      func(context.Context, *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error)
}

func (s *serviceStub) RegisterIdentity(ctx context.Context, input authenticationusecase.RegisterIdentityInput) (authenticationusecase.RegisterIdentityOutput, error) {
	return s.registerIdentity(ctx, input)
}

func (s *serviceStub) LoadContext(ctx context.Context, transaction *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error) {
	return s.loadContext(ctx, transaction)
}
