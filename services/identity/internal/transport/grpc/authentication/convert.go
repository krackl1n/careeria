package authentication

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toGetContextInput(transactionID string) authenticationusecase.GetContextInput {
	id, _ := uuid.Parse(transactionID)
	return authenticationusecase.GetContextInput{AuthorizationTransactionID: id}
}

func toAuthenticateWithPasswordInput(request *identityv1.AuthenticateWithPasswordRequest) authenticationusecase.AuthenticateWithPasswordInput {
	input := authenticationusecase.AuthenticateWithPasswordInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
		Password:                   request.GetPassword(),
	}
	switch login := request.GetLogin().(type) {
	case *identityv1.AuthenticateWithPasswordRequest_Email:
		input.Email = &login.Email
	case *identityv1.AuthenticateWithPasswordRequest_PhoneNumber:
		input.PhoneNumber = &login.PhoneNumber
	}
	return input
}

func toVerifyEmailInput(request *identityv1.VerifyEmailRequest) authenticationusecase.VerifyEmailInput {
	return authenticationusecase.VerifyEmailInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
		Code:                       request.GetCode(),
	}
}

func toVerifyPhoneInput(request *identityv1.VerifyPhoneRequest) authenticationusecase.VerifyPhoneInput {
	return authenticationusecase.VerifyPhoneInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
		Code:                       request.GetCode(),
	}
}

func toSkipMFAEnrollmentInput(request *identityv1.SkipMFAEnrollmentRequest) authenticationusecase.SkipMFAEnrollmentInput {
	return authenticationusecase.SkipMFAEnrollmentInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
	}
}

func toGrantConsentInput(request *identityv1.GrantAuthorizationConsentRequest) authenticationusecase.GrantConsentInput {
	return authenticationusecase.GrantConsentInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
		GrantedScopes:              request.GetGrantedScopes(),
		Remember:                   request.GetRemember(),
	}
}

func toDenyConsentInput(request *identityv1.DenyAuthorizationConsentRequest) authenticationusecase.DenyConsentInput {
	return authenticationusecase.DenyConsentInput{
		AuthorizationTransactionID: parseUUID(request.GetAuthorizationTransactionId()),
	}
}

func parseUUID(value string) uuid.UUID {
	id, _ := uuid.Parse(value)
	return id
}

func ToAuthenticationContext(output authenticationusecase.ContextOutput) (*identityv1.AuthenticationContext, error) {
	transaction := output.AuthorizationTransaction
	client := output.OAuthClient
	if transaction == nil || client == nil {
		return nil, errors.New("convert authentication context: transaction or OAuth client is missing")
	}

	step, err := toAuthenticationStep(output)
	if err != nil {
		return nil, err
	}

	result := &identityv1.AuthenticationContext{
		AuthorizationTransactionId: transaction.ID.UUID().String(),
		OauthClient: &identityv1.OAuthClientSummary{
			ClientId:    string(client.ClientID),
			Name:        client.Name,
			Description: client.Description,
		},
		RequestedScopes: toStrings(transaction.RequestedScopes),
		ExpiresAt:       timestamppb.New(transaction.ExpiresAt),
		NextStep:        step,
	}
	if output.Identity != nil {
		identity, err := toIdentitySummary(output.Identity, output.IdentityProfile)
		if err != nil {
			return nil, err
		}
		result.Identity = identity
	}
	return result, nil
}

func toAuthenticationStep(output authenticationusecase.ContextOutput) (*identityv1.AuthenticationStep, error) {
	transaction := output.AuthorizationTransaction
	switch transaction.NextStep {
	case transactiondomain.NextStepChooseAuthenticationMethod:
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_ChooseAuthenticationMethod{
			ChooseAuthenticationMethod: &identityv1.ChooseAuthenticationMethodStep{AvailableMethods: []string{string(authenticationdomain.MethodPassword)}},
		}}, nil
	case transactiondomain.NextStepMFA:
		methods := toStrings(output.AvailableMFAMethods)
		if len(methods) == 0 {
			return nil, errors.New("convert authentication context: MFA methods are missing")
		}
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_Mfa{
			Mfa: &identityv1.MFAAuthenticationStep{AvailableMethods: methods},
		}}, nil
	case transactiondomain.NextStepEmailVerification:
		verification, err := toEmailVerificationStep(output)
		if err != nil {
			return nil, err
		}
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_EmailVerification{EmailVerification: verification}}, nil
	case transactiondomain.NextStepPhoneVerification:
		verification, err := toPhoneVerificationStep(output)
		if err != nil {
			return nil, err
		}
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_PhoneVerification{PhoneVerification: verification}}, nil
	case transactiondomain.NextStepPasswordChange:
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_PasswordChange{PasswordChange: &identityv1.PasswordChangeStep{}}}, nil
	case transactiondomain.NextStepMFAEnrollment:
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_MfaEnrollment{
			MfaEnrollment: &identityv1.MFAEnrollmentStep{AvailableMethods: []string{string(authenticationdomain.MethodTOTP)}},
		}}, nil
	case transactiondomain.NextStepConsent:
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_Consent{
			Consent: &identityv1.ConsentStep{RequestedScopes: toStrings(transaction.RequestedScopes)},
		}}, nil
	case transactiondomain.NextStepReady:
		return &identityv1.AuthenticationStep{Step: &identityv1.AuthenticationStep_Ready{Ready: &identityv1.AuthorizationReadyStep{}}}, nil
	default:
		return nil, fmt.Errorf("convert authentication context: unsupported next step %q", transaction.NextStep)
	}
}

func toEmailVerificationStep(output authenticationusecase.ContextOutput) (*identityv1.EmailVerificationStep, error) {
	reference := output.AuthorizationTransaction.Verification
	if reference == nil || reference.Channel != transactiondomain.VerificationChannelEmail || output.Identity == nil {
		return nil, errors.New("convert authentication context: email verification state is incomplete")
	}
	for index := range output.Identity.Emails {
		email := &output.Identity.Emails[index]
		if email.HasID(identitydomain.EmailID(reference.ContactID)) {
			return &identityv1.EmailVerificationStep{
				MaskedEmail:        maskEmail(string(email.Address)),
				ChallengeExpiresAt: timestamppb.New(reference.ExpiresAt),
			}, nil
		}
	}
	return nil, errors.New("convert authentication context: verification email is missing")
}

func toPhoneVerificationStep(output authenticationusecase.ContextOutput) (*identityv1.PhoneVerificationStep, error) {
	reference := output.AuthorizationTransaction.Verification
	if reference == nil || reference.Channel != transactiondomain.VerificationChannelPhone || output.Identity == nil {
		return nil, errors.New("convert authentication context: phone verification state is incomplete")
	}
	for index := range output.Identity.Phones {
		phone := &output.Identity.Phones[index]
		if phone.HasID(identitydomain.PhoneID(reference.ContactID)) {
			return &identityv1.PhoneVerificationStep{
				MaskedPhoneNumber:  maskPhone(string(phone.Number)),
				ChallengeExpiresAt: timestamppb.New(reference.ExpiresAt),
			}, nil
		}
	}
	return nil, errors.New("convert authentication context: verification phone is missing")
}

func toIdentitySummary(identity *identitydomain.Identity, profile *identityprofiledomain.Profile) (*identityv1.IdentitySummary, error) {
	if profile == nil {
		return nil, errors.New("convert authentication context: identity profile is missing")
	}
	result := &identityv1.IdentitySummary{
		Subject: identity.ID.UUID().String(),
		Profile: &identityv1.IdentityProfile{
			FamilyName:     profile.FamilyName,
			GivenName:      profile.GivenName,
			MiddleName:     profile.MiddleName,
			Source:         toProfileSource(profile.Source),
			AssuranceLevel: toAssuranceLevel(profile.AssuranceLevel),
			Version:        profile.Version,
			CreatedAt:      timestamppb.New(profile.CreatedAt),
			UpdatedAt:      timestamppb.New(profile.UpdatedAt),
		},
	}
	for index := range identity.Emails {
		email := &identity.Emails[index]
		if email.IsPrimaryEmail() {
			address := string(email.Address)
			verified := email.VerifiedAt != nil
			result.Email = &address
			result.EmailVerified = &verified
			break
		}
	}
	for index := range identity.Phones {
		phone := &identity.Phones[index]
		if phone.IsPrimaryPhone() {
			number := string(phone.Number)
			verified := phone.VerifiedAt != nil
			result.PhoneNumber = &number
			result.PhoneNumberVerified = &verified
			break
		}
	}
	return result, nil
}

func toProfileSource(source identityprofiledomain.Source) identityv1.IdentityProfileSource {
	switch source {
	case identityprofiledomain.SourceSelfDeclared:
		return identityv1.IdentityProfileSource_IDENTITY_PROFILE_SOURCE_SELF_DECLARED
	case identityprofiledomain.SourceProvisioned:
		return identityv1.IdentityProfileSource_IDENTITY_PROFILE_SOURCE_PROVISIONED
	case identityprofiledomain.SourceVerified:
		return identityv1.IdentityProfileSource_IDENTITY_PROFILE_SOURCE_VERIFIED
	default:
		return identityv1.IdentityProfileSource_IDENTITY_PROFILE_SOURCE_UNSPECIFIED
	}
}

func toAssuranceLevel(level identityprofiledomain.AssuranceLevel) identityv1.IdentityAssuranceLevel {
	switch level {
	case identityprofiledomain.AssuranceLevelSelfDeclared:
		return identityv1.IdentityAssuranceLevel_IDENTITY_ASSURANCE_LEVEL_SELF_DECLARED
	case identityprofiledomain.AssuranceLevelOrganizationAsserted:
		return identityv1.IdentityAssuranceLevel_IDENTITY_ASSURANCE_LEVEL_ORGANIZATION_ASSERTED
	case identityprofiledomain.AssuranceLevelIdentityVerified:
		return identityv1.IdentityAssuranceLevel_IDENTITY_ASSURANCE_LEVEL_IDENTITY_VERIFIED
	default:
		return identityv1.IdentityAssuranceLevel_IDENTITY_ASSURANCE_LEVEL_UNSPECIFIED
	}
}

func toStrings[Value ~string](values []Value) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func maskEmail(value string) string {
	local, domain, ok := strings.Cut(value, "@")
	if !ok || local == "" || domain == "" {
		return "***"
	}
	return string([]rune(local)[0]) + "***@" + domain
}

func maskPhone(value string) string {
	runes := []rune(value)
	const visible = 4
	if len(runes) <= visible {
		return strings.Repeat("*", len(runes))
	}
	prefix := ""
	maskLength := len(runes) - visible
	if runes[0] == '+' {
		prefix = "+"
		maskLength--
	}
	return prefix + strings.Repeat("*", maskLength) + string(runes[len(runes)-visible:])
}
