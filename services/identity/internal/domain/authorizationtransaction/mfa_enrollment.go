package authorizationtransaction

import (
	"fmt"
	"time"
)

func (t *AuthorizationTransaction) SkipMFAEnrollment(now time.Time) error {
	if err := t.completeRequiredStep(NextStepMFAEnrollment, NextStepConsent, now); err != nil {
		return fmt.Errorf("skip MFA enrollment: %w", err)
	}
	return nil
}
