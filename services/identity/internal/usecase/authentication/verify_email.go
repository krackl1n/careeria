package authentication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
)

type VerifyEmailInput struct {
	AuthorizationTransactionID uuid.UUID
	Code                       string
	ConsumedFromIP             *sessiondomain.IPAddress
}

type VerifyEmailOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

func (s *Service) VerifyEmail(ctx context.Context, input VerifyEmailInput) (VerifyEmailOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil || !isSixDigitVerificationCode(input.Code) {
		return VerifyEmailOutput{}, apperr.New(apperr.TypeInvalidArgument, "contact_verification_invalid_input", "email verification request is invalid", nil)
	}

	now := s.clock.Now()
	var output VerifyEmailOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactiondomain.ID(input.AuthorizationTransactionID))
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		emailID, err := s.contactVerifications.VerifyEmail(
			txCtx,
			transaction,
			input.Code,
			input.ConsumedFromIP,
			now,
		)
		if err != nil {
			if errors.Is(err, contactverificationservice.ErrChallengeUnavailable) {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "email verification challenge is not available", err)
			}
			return fmt.Errorf("verify email challenge: %w", err)
		}

		identity, err := s.identities.Get(txCtx, *transaction.IdentityID)
		if err != nil {
			if errors.Is(err, identitydomain.ErrNotFound) {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "email verification identity is not available", err)
			}
			return fmt.Errorf("get email verification identity: %w", err)
		}
		if identity.Status != identitydomain.StatusPending && identity.Status != identitydomain.StatusActive {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "email verification identity is not available", nil)
		}

		if err := identity.VerifyEmail(emailID, now); err != nil {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "email cannot be verified", err)
		}
		if identity.Status != identitydomain.StatusActive {
			if err := identity.Activate(now); err != nil {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "identity cannot be activated", err)
			}
		}
		if err := s.identities.Update(txCtx, identity); err != nil {
			return fmt.Errorf("update verified identity email: %w", err)
		}

		decision, err := s.resolveNextStep(txCtx, transaction, identity.ID, flowdomain.Requirements{}, now)
		if err != nil {
			return fmt.Errorf("resolve next step after email verification: %w", err)
		}
		if err := transaction.CompleteEmailVerification(decision.NextStep, now); err != nil {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "email verification cannot be completed", err)
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction after email verification: %w", err)
		}

		output = VerifyEmailOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return VerifyEmailOutput{}, err
	}

	return output, nil
}

func isSixDigitVerificationCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
