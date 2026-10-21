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

type VerifyPhoneInput struct {
	AuthorizationTransactionID uuid.UUID
	Code                       string
	ConsumedFromIP             *sessiondomain.IPAddress
}

type VerifyPhoneOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

func (s *Service) VerifyPhone(ctx context.Context, input VerifyPhoneInput) (VerifyPhoneOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil || !isSixDigitVerificationCode(input.Code) {
		return VerifyPhoneOutput{}, apperr.New(apperr.TypeInvalidArgument, "contact_verification_invalid_input", "phone verification request is invalid", nil)
	}

	now := s.clock.Now()
	var output VerifyPhoneOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactiondomain.ID(input.AuthorizationTransactionID))
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		phoneID, err := s.contactVerifications.VerifyPhone(
			txCtx,
			transaction,
			input.Code,
			input.ConsumedFromIP,
			now,
		)
		if err != nil {
			if errors.Is(err, contactverificationservice.ErrChallengeUnavailable) {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "phone verification challenge is not available", err)
			}
			return fmt.Errorf("verify phone challenge: %w", err)
		}

		identity, err := s.identities.Get(txCtx, *transaction.IdentityID)
		if err != nil {
			if errors.Is(err, identitydomain.ErrNotFound) {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "phone verification identity is not available", err)
			}
			return fmt.Errorf("get phone verification identity: %w", err)
		}
		if identity.Status != identitydomain.StatusPending && identity.Status != identitydomain.StatusActive {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "phone verification identity is not available", nil)
		}

		if err := identity.VerifyPhone(phoneID, now); err != nil {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "phone cannot be verified", err)
		}
		if identity.Status != identitydomain.StatusActive {
			if err := identity.Activate(now); err != nil {
				return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "identity cannot be activated", err)
			}
		}
		if err := s.identities.Update(txCtx, identity); err != nil {
			return fmt.Errorf("update verified identity phone: %w", err)
		}

		decision, err := s.resolveNextStep(txCtx, transaction, identity.ID, flowdomain.Requirements{}, now)
		if err != nil {
			return fmt.Errorf("resolve next step after phone verification: %w", err)
		}
		if err := transaction.CompletePhoneVerification(decision.NextStep, now); err != nil {
			return apperr.New(apperr.TypeConflict, "contact_verification_invalid_state", "phone verification cannot be completed", err)
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction after phone verification: %w", err)
		}

		output = VerifyPhoneOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return VerifyPhoneOutput{}, err
	}

	return output, nil
}
