package authentication

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

func TestGetAuthenticationMethods(t *testing.T) {
	response, err := NewAuthenticationHandler(&serviceStub{}).GetAuthenticationMethods(t.Context(), &identityv1.GetAuthenticationMethodsRequest{})

	require.NoError(t, err)
	require.NoError(t, response.ValidateAll())
	require.Equal(t, "password", response.GetMethods()[0].GetCode())
}

func TestVerifyEmailUsesReturnedTransaction(t *testing.T) {
	transactionID := uuid.New()
	transaction := transportContext(t, transactionID).AuthorizationTransaction
	verifyCalled := false
	contextCalled := false
	service := &serviceStub{
		verifyEmail: func(_ context.Context, input authenticationusecase.VerifyEmailInput) (authenticationusecase.VerifyEmailOutput, error) {
			verifyCalled = true
			require.Equal(t, transactionID, input.AuthorizationTransactionID)
			require.Equal(t, "123456", input.Code)
			return authenticationusecase.VerifyEmailOutput{AuthorizationTransaction: transaction}, nil
		},
		loadContext: func(_ context.Context, actual *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error) {
			contextCalled = true
			require.Same(t, transaction, actual)
			return transportContext(t, transactionID), nil
		},
	}

	response, err := NewAuthenticationHandler(service).VerifyEmail(t.Context(), &identityv1.VerifyEmailRequest{
		AuthorizationTransactionId: transactionID.String(),
		Code:                       "123456",
	})

	require.NoError(t, err)
	require.True(t, verifyCalled)
	require.True(t, contextCalled)
	require.NoError(t, response.ValidateAll())
	require.Equal(t, transactionID.String(), response.GetContext().GetAuthorizationTransactionId())
}

func TestSkipMFAEnrollmentReturnsFreshContext(t *testing.T) {
	transactionID := uuid.New()
	transaction := transportContext(t, transactionID).AuthorizationTransaction
	service := &serviceStub{
		skipMFAEnrollment: func(_ context.Context, input authenticationusecase.SkipMFAEnrollmentInput) (authenticationusecase.SkipMFAEnrollmentOutput, error) {
			require.Equal(t, transactionID, input.AuthorizationTransactionID)
			return authenticationusecase.SkipMFAEnrollmentOutput{AuthorizationTransaction: transaction}, nil
		},
		loadContext: func(context.Context, *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error) {
			return transportContext(t, transactionID), nil
		},
	}

	response, err := NewAuthenticationHandler(service).SkipMFAEnrollment(t.Context(), &identityv1.SkipMFAEnrollmentRequest{
		AuthorizationTransactionId: transactionID.String(),
	})

	require.NoError(t, err)
	require.NoError(t, response.ValidateAll())
}

func transportContext(t *testing.T, transactionID uuid.UUID) authenticationusecase.ContextOutput {
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
	getContext               func(context.Context, authenticationusecase.GetContextInput) (authenticationusecase.ContextOutput, error)
	loadContext              func(context.Context, *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error)
	authenticateWithPassword func(context.Context, authenticationusecase.AuthenticateWithPasswordInput) (authenticationusecase.AuthenticateWithPasswordOutput, error)
	verifyEmail              func(context.Context, authenticationusecase.VerifyEmailInput) (authenticationusecase.VerifyEmailOutput, error)
	verifyPhone              func(context.Context, authenticationusecase.VerifyPhoneInput) (authenticationusecase.VerifyPhoneOutput, error)
	skipMFAEnrollment        func(context.Context, authenticationusecase.SkipMFAEnrollmentInput) (authenticationusecase.SkipMFAEnrollmentOutput, error)
	grantConsent             func(context.Context, authenticationusecase.GrantConsentInput) (authenticationusecase.GrantConsentOutput, error)
	denyConsent              func(context.Context, authenticationusecase.DenyConsentInput) (authenticationusecase.DenyConsentOutput, error)
}

func (s *serviceStub) GetContext(ctx context.Context, input authenticationusecase.GetContextInput) (authenticationusecase.ContextOutput, error) {
	return s.getContext(ctx, input)
}

func (s *serviceStub) LoadContext(ctx context.Context, transaction *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error) {
	return s.loadContext(ctx, transaction)
}

func (s *serviceStub) AuthenticateWithPassword(ctx context.Context, input authenticationusecase.AuthenticateWithPasswordInput) (authenticationusecase.AuthenticateWithPasswordOutput, error) {
	return s.authenticateWithPassword(ctx, input)
}

func (s *serviceStub) VerifyEmail(ctx context.Context, input authenticationusecase.VerifyEmailInput) (authenticationusecase.VerifyEmailOutput, error) {
	return s.verifyEmail(ctx, input)
}

func (s *serviceStub) VerifyPhone(ctx context.Context, input authenticationusecase.VerifyPhoneInput) (authenticationusecase.VerifyPhoneOutput, error) {
	return s.verifyPhone(ctx, input)
}

func (s *serviceStub) SkipMFAEnrollment(ctx context.Context, input authenticationusecase.SkipMFAEnrollmentInput) (authenticationusecase.SkipMFAEnrollmentOutput, error) {
	return s.skipMFAEnrollment(ctx, input)
}

func (s *serviceStub) GrantConsent(ctx context.Context, input authenticationusecase.GrantConsentInput) (authenticationusecase.GrantConsentOutput, error) {
	return s.grantConsent(ctx, input)
}

func (s *serviceStub) DenyConsent(ctx context.Context, input authenticationusecase.DenyConsentInput) (authenticationusecase.DenyConsentOutput, error) {
	return s.denyConsent(ctx, input)
}
