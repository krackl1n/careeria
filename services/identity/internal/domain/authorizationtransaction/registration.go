package authorizationtransaction

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (t *AuthorizationTransaction) BeginPasswordRegistration(
	identityID identitydomain.ID,
	verificationStep NextStep,
	now time.Time,
) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("begin password registration in authorization transaction: %w", err)
	}
	if t.Status != StatusPending {
		return fmt.Errorf("begin password registration in authorization transaction: %w", ErrInvalidStatus)
	}
	if t.NextStep != NextStepChooseAuthenticationMethod || t.IdentityID != nil || len(t.AuthenticatedMethods) != 0 {
		return fmt.Errorf("begin password registration in authorization transaction: %w", ErrInvalidStep)
	}
	if !identityID.IsValid() {
		return errors.New("begin password registration in authorization transaction: identity identifier is invalid")
	}
	if verificationStep != NextStepEmailVerification && verificationStep != NextStepPhoneVerification {
		return fmt.Errorf("begin password registration in authorization transaction: %w", ErrInvalidStep)
	}
	if err := t.transitionTo(verificationStep); err != nil {
		return fmt.Errorf("begin password registration in authorization transaction: %w", err)
	}

	t.IdentityID = domain.ClonePointer(&identityID)
	t.AuthenticatedMethods = []authenticationdomain.Method{authenticationdomain.MethodPassword}
	return nil
}
