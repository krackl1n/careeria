package authentication

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
)

type AuthenticationHandler struct {
	identityv1.UnimplementedAuthenticationServiceServer

	authentication AuthenticationService
}

type AuthenticationService interface {
	contextLoader
	GetContext(context.Context, authenticationusecase.GetContextInput) (authenticationusecase.ContextOutput, error)
	AuthenticateWithPassword(context.Context, authenticationusecase.AuthenticateWithPasswordInput) (authenticationusecase.AuthenticateWithPasswordOutput, error)
	VerifyEmail(context.Context, authenticationusecase.VerifyEmailInput) (authenticationusecase.VerifyEmailOutput, error)
	VerifyPhone(context.Context, authenticationusecase.VerifyPhoneInput) (authenticationusecase.VerifyPhoneOutput, error)
	SkipMFAEnrollment(context.Context, authenticationusecase.SkipMFAEnrollmentInput) (authenticationusecase.SkipMFAEnrollmentOutput, error)
	GrantConsent(context.Context, authenticationusecase.GrantConsentInput) (authenticationusecase.GrantConsentOutput, error)
	DenyConsent(context.Context, authenticationusecase.DenyConsentInput) (authenticationusecase.DenyConsentOutput, error)
}

var _ identityv1.AuthenticationServiceServer = (*AuthenticationHandler)(nil)

func NewAuthenticationHandler(authentication AuthenticationService) *AuthenticationHandler {
	return &AuthenticationHandler{authentication: authentication}
}
