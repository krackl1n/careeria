package authorizationtransaction

type NextStep string

const (
	NextStepChooseAuthenticationMethod NextStep = "choose_authentication_method"
	NextStepPasswordChange             NextStep = "password_change"
	NextStepEmailVerification          NextStep = "email_verification"
	NextStepPhoneVerification          NextStep = "phone_verification"
	NextStepMFAEnrollment              NextStep = "mfa_enrollment"
	NextStepMFA                        NextStep = "mfa"
	NextStepConsent                    NextStep = "consent"
	NextStepReady                      NextStep = "ready"
)

func (s NextStep) CanTransitionTo(next NextStep) bool {
	switch s {
	case NextStepChooseAuthenticationMethod:
		return next == NextStepPasswordChange ||
			next == NextStepEmailVerification ||
			next == NextStepPhoneVerification ||
			next == NextStepMFAEnrollment ||
			next == NextStepMFA ||
			next == NextStepConsent
	case NextStepPasswordChange:
		return next == NextStepEmailVerification ||
			next == NextStepPhoneVerification ||
			next == NextStepMFAEnrollment ||
			next == NextStepMFA ||
			next == NextStepConsent
	case NextStepEmailVerification:
		return next == NextStepPhoneVerification ||
			next == NextStepMFAEnrollment ||
			next == NextStepMFA ||
			next == NextStepConsent
	case NextStepPhoneVerification:
		return next == NextStepMFAEnrollment ||
			next == NextStepMFA ||
			next == NextStepConsent
	case NextStepMFAEnrollment:
		return next == NextStepConsent
	case NextStepMFA:
		return next == NextStepConsent
	case NextStepConsent:
		return next == NextStepReady
	default:
		return false
	}
}
