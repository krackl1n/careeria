package authorizationtransaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextStepTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current NextStep
		next    NextStep
		allowed bool
	}{
		{name: "authentication to password change", current: NextStepChooseAuthenticationMethod, next: NextStepPasswordChange, allowed: true},
		{name: "authentication to email verification", current: NextStepChooseAuthenticationMethod, next: NextStepEmailVerification, allowed: true},
		{name: "email verification to MFA", current: NextStepEmailVerification, next: NextStepMFA, allowed: true},
		{name: "email verification to MFA enrollment", current: NextStepEmailVerification, next: NextStepMFAEnrollment, allowed: true},
		{name: "email verification to phone verification", current: NextStepEmailVerification, next: NextStepPhoneVerification, allowed: true},
		{name: "phone verification to MFA enrollment", current: NextStepPhoneVerification, next: NextStepMFAEnrollment, allowed: true},
		{name: "MFA enrollment to consent", current: NextStepMFAEnrollment, next: NextStepConsent, allowed: true},
		{name: "MFA to consent", current: NextStepMFA, next: NextStepConsent, allowed: true},
		{name: "consent to ready", current: NextStepConsent, next: NextStepReady, allowed: true},
		{name: "email verification cannot return to authentication", current: NextStepEmailVerification, next: NextStepChooseAuthenticationMethod},
		{name: "phone verification cannot return to email verification", current: NextStepPhoneVerification, next: NextStepEmailVerification},
		{name: "MFA enrollment cannot require MFA immediately", current: NextStepMFAEnrollment, next: NextStepMFA},
		{name: "MFA enrollment cannot return to phone verification", current: NextStepMFAEnrollment, next: NextStepPhoneVerification},
		{name: "consent cannot return to MFA", current: NextStepConsent, next: NextStepMFA},
		{name: "ready is terminal step", current: NextStepReady, next: NextStepConsent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.allowed, test.current.CanTransitionTo(test.next))
		})
	}
}
