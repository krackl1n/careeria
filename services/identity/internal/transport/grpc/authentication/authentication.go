package authentication

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
)

func (h *AuthenticationHandler) GetAuthenticationMethods(
	context.Context,
	*identityv1.GetAuthenticationMethodsRequest,
) (*identityv1.GetAuthenticationMethodsResponse, error) {
	return &identityv1.GetAuthenticationMethodsResponse{Methods: []*identityv1.AuthenticationMethod{{
		Code:        "password",
		DisplayName: "Пароль",
		Description: stringPointer("Вход по email или номеру телефона и паролю"),
	}}}, nil
}

func (h *AuthenticationHandler) GetAuthenticationContext(
	ctx context.Context,
	request *identityv1.GetAuthenticationContextRequest,
) (*identityv1.GetAuthenticationContextResponse, error) {
	output, err := h.authentication.GetContext(ctx, toGetContextInput(request.GetAuthorizationTransactionId()))
	if err != nil {
		return nil, err
	}
	contextValue, err := ToAuthenticationContext(output)
	if err != nil {
		return nil, err
	}
	return &identityv1.GetAuthenticationContextResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) AuthenticateWithPassword(
	ctx context.Context,
	request *identityv1.AuthenticateWithPasswordRequest,
) (*identityv1.AuthenticateWithPasswordResponse, error) {
	output, err := h.authentication.AuthenticateWithPassword(ctx, toAuthenticateWithPasswordInput(request))
	if err != nil {
		return nil, err
	}
	contextValue, err := contextForTransaction(ctx, h.authentication, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	return &identityv1.AuthenticateWithPasswordResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) VerifyEmail(
	ctx context.Context,
	request *identityv1.VerifyEmailRequest,
) (*identityv1.VerifyEmailResponse, error) {
	input := toVerifyEmailInput(request)
	output, err := h.authentication.VerifyEmail(ctx, input)
	if err != nil {
		return nil, err
	}
	contextValue, err := contextForTransaction(ctx, h.authentication, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	return &identityv1.VerifyEmailResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) VerifyPhone(
	ctx context.Context,
	request *identityv1.VerifyPhoneRequest,
) (*identityv1.VerifyPhoneResponse, error) {
	input := toVerifyPhoneInput(request)
	output, err := h.authentication.VerifyPhone(ctx, input)
	if err != nil {
		return nil, err
	}
	contextValue, err := contextForTransaction(ctx, h.authentication, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	return &identityv1.VerifyPhoneResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) SkipMFAEnrollment(
	ctx context.Context,
	request *identityv1.SkipMFAEnrollmentRequest,
) (*identityv1.SkipMFAEnrollmentResponse, error) {
	output, err := h.authentication.SkipMFAEnrollment(ctx, toSkipMFAEnrollmentInput(request))
	if err != nil {
		return nil, err
	}
	contextValue, err := contextForTransaction(ctx, h.authentication, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	return &identityv1.SkipMFAEnrollmentResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) GrantAuthorizationConsent(
	ctx context.Context,
	request *identityv1.GrantAuthorizationConsentRequest,
) (*identityv1.GrantAuthorizationConsentResponse, error) {
	output, err := h.authentication.GrantConsent(ctx, toGrantConsentInput(request))
	if err != nil {
		return nil, err
	}
	contextValue, err := contextForTransaction(ctx, h.authentication, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	return &identityv1.GrantAuthorizationConsentResponse{Context: contextValue}, nil
}

func (h *AuthenticationHandler) DenyAuthorizationConsent(
	ctx context.Context,
	request *identityv1.DenyAuthorizationConsentRequest,
) (*identityv1.DenyAuthorizationConsentResponse, error) {
	output, err := h.authentication.DenyConsent(ctx, toDenyConsentInput(request))
	if err != nil {
		return nil, err
	}
	return &identityv1.DenyAuthorizationConsentResponse{RedirectUri: output.RedirectURI}, nil
}

func stringPointer(value string) *string {
	return &value
}
