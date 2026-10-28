package authenticationflow

import (
	"testing"

	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestResolveNextStepUsesRequirementPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      Input
		wantStep   transactiondomain.NextStep
		wantSource *transactiondomain.AuthorizationSource
	}{
		{
			name: "password change has highest priority",
			input: Input{
				CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
				Requirements: Requirements{
					PasswordChangeRequired: true,
					ContactVerification:    ContactVerificationEmail,
				},
				Facts: factsPointer(mustFacts(t, []authenticationdomain.Method{authenticationdomain.MethodTOTP}, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepPasswordChange,
		},
		{
			name: "email verification precedes MFA",
			input: Input{
				CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
				Requirements: Requirements{
					ContactVerification: ContactVerificationEmail,
				},
				Facts: factsPointer(mustFacts(t, []authenticationdomain.Method{authenticationdomain.MethodTOTP}, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepEmailVerification,
		},
		{
			name: "phone verification follows password change",
			input: Input{
				CurrentStep: transactiondomain.NextStepPasswordChange,
				Requirements: Requirements{
					ContactVerification: ContactVerificationPhone,
				},
				Facts: factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepPhoneVerification,
		},
		{
			name: "MFA enrollment is offered without a factor",
			input: Input{
				CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
				Facts:       factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepMFAEnrollment,
		},
		{
			name: "configured MFA must be completed",
			input: Input{
				CurrentStep: transactiondomain.NextStepEmailVerification,
				Facts:       factsPointer(mustFacts(t, []authenticationdomain.Method{authenticationdomain.MethodWebAuthn}, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepMFA,
		},
		{
			name: "required consent is interactive after skipped enrollment",
			input: Input{
				CurrentStep: transactiondomain.NextStepMFAEnrollment,
				Facts:       factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeRequired, false)),
			},
			wantStep: transactiondomain.NextStepConsent,
		},
		{
			name: "uncovered remembered grant requires consent",
			input: Input{
				CurrentStep: transactiondomain.NextStepMFA,
				Facts:       factsPointer(mustFacts(t, []authenticationdomain.Method{authenticationdomain.MethodTOTP}, oauthclientdomain.ConsentModeRemember, false)),
			},
			wantStep: transactiondomain.NextStepConsent,
		},
		{
			name: "remembered grant authorizes automatically",
			input: Input{
				CurrentStep: transactiondomain.NextStepMFA,
				Facts:       factsPointer(mustFacts(t, []authenticationdomain.Method{authenticationdomain.MethodTOTP}, oauthclientdomain.ConsentModeRemember, true)),
			},
			wantStep:   transactiondomain.NextStepConsent,
			wantSource: sourcePointer(transactiondomain.AuthorizationSourceRememberedGrant),
		},
		{
			name: "trusted client authorizes automatically",
			input: Input{
				CurrentStep: transactiondomain.NextStepMFAEnrollment,
				Facts:       factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeTrusted, false)),
			},
			wantStep:   transactiondomain.NextStepConsent,
			wantSource: sourcePointer(transactiondomain.AuthorizationSourceTrustedClient),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decision, err := ResolveNextStep(test.input)
			require.NoError(t, err)
			require.Equal(t, test.wantStep, decision.NextStep)
			require.Equal(t, test.wantSource, decision.AuthorizationSource)
		})
	}
}

func TestResolveNextStepRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := ResolveNextStep(Input{
		CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
		Requirements: Requirements{
			ContactVerification: ContactVerificationRequirement("unknown"),
		},
		Facts: factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeRemember, false)),
	})
	require.Error(t, err)

	_, err = ResolveNextStep(Input{
		CurrentStep: transactiondomain.NextStepReady,
		Facts:       factsPointer(mustFacts(t, nil, oauthclientdomain.ConsentModeRemember, false)),
	})
	require.ErrorIs(t, err, ErrNoNextStep)
}

func TestResolveNextStepDoesNotRequireFactsForEarlierRequirements(t *testing.T) {
	t.Parallel()

	decision, err := ResolveNextStep(Input{
		CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
		Requirements: Requirements{
			PasswordChangeRequired: true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, transactiondomain.NextStepPasswordChange, decision.NextStep)

	decision, err = ResolveNextStep(Input{
		CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod,
		Requirements: Requirements{
			ContactVerification: ContactVerificationEmail,
		},
	})
	require.NoError(t, err)
	require.Equal(t, transactiondomain.NextStepEmailVerification, decision.NextStep)

	_, err = ResolveNextStep(Input{CurrentStep: transactiondomain.NextStepChooseAuthenticationMethod})
	require.ErrorIs(t, err, ErrFactsRequired)
}

func mustFacts(
	t *testing.T,
	methods []authenticationdomain.Method,
	mode oauthclientdomain.ConsentMode,
	covers bool,
) Facts {
	t.Helper()
	facts, err := NewFacts(methods, mode, covers)
	require.NoError(t, err)
	return facts
}

func sourcePointer(source transactiondomain.AuthorizationSource) *transactiondomain.AuthorizationSource {
	return &source
}

func factsPointer(facts Facts) *Facts {
	return &facts
}
