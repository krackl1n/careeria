package authentication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

type SkipMFAEnrollmentInput struct {
	AuthorizationTransactionID uuid.UUID
}

type SkipMFAEnrollmentOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

func (s *Service) SkipMFAEnrollment(ctx context.Context, input SkipMFAEnrollmentInput) (SkipMFAEnrollmentOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return SkipMFAEnrollmentOutput{}, apperr.New(apperr.TypeInvalidArgument, "mfa_enrollment_invalid_input", "MFA enrollment request is invalid", nil)
	}

	now := s.clock.Now()
	var output SkipMFAEnrollmentOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactiondomain.ID(input.AuthorizationTransactionID))
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		if transaction.IdentityID == nil {
			return apperr.New(apperr.TypeConflict, "mfa_enrollment_invalid_state", "MFA enrollment cannot be skipped", errors.New("authorization transaction identity is missing"))
		}
		decision, err := s.resolveNextStep(txCtx, transaction, *transaction.IdentityID, flowdomain.Requirements{}, now)
		if err != nil {
			return fmt.Errorf("resolve next step after skipped MFA enrollment: %w", err)
		}
		if decision.NextStep != transactiondomain.NextStepConsent {
			return fmt.Errorf("skip MFA enrollment: unexpected next step %q", decision.NextStep)
		}

		if err := transaction.SkipMFAEnrollment(now); err != nil {
			return apperr.New(apperr.TypeConflict, "mfa_enrollment_invalid_state", "MFA enrollment cannot be skipped", err)
		}
		if decision.AuthorizationSource != nil {
			if err := transaction.AuthorizeScopes(transaction.RequestedScopes, *decision.AuthorizationSource, now); err != nil {
				return fmt.Errorf("automatically authorize scopes after skipped MFA enrollment: %w", err)
			}
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction after skipped MFA enrollment: %w", err)
		}

		output = SkipMFAEnrollmentOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return SkipMFAEnrollmentOutput{}, err
	}

	return output, nil
}
