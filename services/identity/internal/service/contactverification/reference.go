package contactverification

import (
	"fmt"
	"time"

	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

func emailVerificationReference(
	transaction *transactiondomain.AuthorizationTransaction,
	now time.Time,
) (transactiondomain.VerificationReference, error) {
	return verificationReference(
		transaction,
		transactiondomain.NextStepEmailVerification,
		transactiondomain.VerificationChannelEmail,
		now,
	)
}

func phoneVerificationReference(
	transaction *transactiondomain.AuthorizationTransaction,
	now time.Time,
) (transactiondomain.VerificationReference, error) {
	return verificationReference(
		transaction,
		transactiondomain.NextStepPhoneVerification,
		transactiondomain.VerificationChannelPhone,
		now,
	)
}

func verificationReference(
	transaction *transactiondomain.AuthorizationTransaction,
	step transactiondomain.NextStep,
	channel transactiondomain.VerificationChannel,
	now time.Time,
) (transactiondomain.VerificationReference, error) {
	if transaction == nil ||
		transaction.Status != transactiondomain.StatusPending ||
		transaction.NextStep != step ||
		transaction.IdentityID == nil ||
		!transaction.IdentityID.IsValid() ||
		transaction.AuthenticationLevel != nil ||
		transaction.Verification == nil ||
		transaction.Verification.Channel != channel ||
		!now.Before(transaction.ExpiresAt) ||
		!now.Before(transaction.Verification.ExpiresAt) {
		return transactiondomain.VerificationReference{}, fmt.Errorf("get verification reference: %w", ErrChallengeUnavailable)
	}
	if _, err := transactiondomain.NewVerificationReference(
		transaction.Verification.Channel,
		transaction.Verification.ContactID,
		transaction.Verification.ChallengeID,
		transaction.Verification.ExpiresAt,
	); err != nil {
		return transactiondomain.VerificationReference{}, fmt.Errorf("get verification reference: %w", ErrChallengeUnavailable)
	}

	return *transaction.Verification, nil
}
