package authenticationflow

import (
	"errors"
	"fmt"

	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type Input struct {
	CurrentStep  transactiondomain.NextStep
	Requirements Requirements
	Facts        *Facts
}

type Decision struct {
	NextStep            transactiondomain.NextStep
	AuthorizationSource *transactiondomain.AuthorizationSource
}

func ResolveNextStep(input Input) (Decision, error) {
	if err := input.Requirements.Validate(); err != nil {
		return Decision{}, fmt.Errorf("resolve authentication flow next step: %w", err)
	}
	if input.Requirements.PasswordChangeRequired && input.CurrentStep.CanTransitionTo(transactiondomain.NextStepPasswordChange) {
		return Decision{NextStep: transactiondomain.NextStepPasswordChange}, nil
	}

	verificationStep := verificationNextStep(input.Requirements.ContactVerification)
	if verificationStep != "" && input.CurrentStep.CanTransitionTo(verificationStep) {
		return Decision{NextStep: verificationStep}, nil
	}

	if input.Facts == nil {
		return Decision{}, fmt.Errorf("resolve authentication flow next step: %w", ErrFactsRequired)
	}

	mfaStep := transactiondomain.NextStepMFAEnrollment
	if input.Facts.HasUsableMFA() {
		mfaStep = transactiondomain.NextStepMFA
	}
	if input.CurrentStep.CanTransitionTo(mfaStep) {
		return Decision{NextStep: mfaStep}, nil
	}

	if !input.CurrentStep.CanTransitionTo(transactiondomain.NextStepConsent) {
		return Decision{}, fmt.Errorf("resolve authentication flow next step from %q: %w", input.CurrentStep, ErrNoNextStep)
	}

	switch input.Facts.consentMode {
	case oauthclientdomain.ConsentModeRequired:
		return Decision{NextStep: transactiondomain.NextStepConsent}, nil
	case oauthclientdomain.ConsentModeRemember:
		if !input.Facts.rememberedGrantCovers {
			return Decision{NextStep: transactiondomain.NextStepConsent}, nil
		}
		source := transactiondomain.AuthorizationSourceRememberedGrant
		return Decision{NextStep: transactiondomain.NextStepConsent, AuthorizationSource: &source}, nil
	case oauthclientdomain.ConsentModeTrusted:
		source := transactiondomain.AuthorizationSourceTrustedClient
		return Decision{NextStep: transactiondomain.NextStepConsent, AuthorizationSource: &source}, nil
	default:
		return Decision{}, errors.New("resolve authentication flow next step: consent mode is invalid")
	}
}

func verificationNextStep(requirement ContactVerificationRequirement) transactiondomain.NextStep {
	switch requirement {
	case ContactVerificationEmail:
		return transactiondomain.NextStepEmailVerification
	case ContactVerificationPhone:
		return transactiondomain.NextStepPhoneVerification
	default:
		return ""
	}
}
