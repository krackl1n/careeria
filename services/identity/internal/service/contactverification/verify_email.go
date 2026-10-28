package contactverification

import (
	"context"
	"errors"
	"fmt"
	"time"

	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	emailchallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/emailverificationchallenge"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (s *Service) VerifyEmail(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	presentedCode string,
	verifiedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (identitydomain.EmailID, error) {
	if !isSixDigitCode(presentedCode) || now.IsZero() {
		return identitydomain.EmailID{}, errors.New("verify email challenge: input is invalid")
	}

	reference, err := emailVerificationReference(transaction, now)
	if err != nil {
		return identitydomain.EmailID{}, err
	}
	challengeID := emailchallengedomain.ID(reference.ChallengeID)
	emailID := identitydomain.EmailID(reference.ContactID)

	challenge, err := s.emailChallenges.GetForUpdate(ctx, challengeID)
	if err != nil {
		if errors.Is(err, emailchallengedomain.ErrNotFound) {
			return identitydomain.EmailID{}, fmt.Errorf("verify email challenge: %w", ErrChallengeUnavailable)
		}
		return identitydomain.EmailID{}, fmt.Errorf("verify email challenge: get challenge: %w", err)
	}
	if challenge.EmailID != emailID {
		return identitydomain.EmailID{}, fmt.Errorf("verify email challenge: %w", ErrChallengeUnavailable)
	}
	if !s.tokenMatches(presentedCode, string(challenge.TokenHash)) {
		return identitydomain.EmailID{}, errors.New("verify email challenge: code does not match")
	}
	if err := challenge.Consume(verifiedFromIP, now); err != nil {
		return identitydomain.EmailID{}, fmt.Errorf("verify email challenge: %w: %v", ErrChallengeUnavailable, err)
	}
	if err := s.emailChallenges.Update(ctx, challenge); err != nil {
		return identitydomain.EmailID{}, fmt.Errorf("verify email challenge: save challenge: %w", err)
	}

	return emailID, nil
}
