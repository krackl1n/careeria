package contactverification

import (
	"context"
	"errors"
	"fmt"
	"time"

	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	phonechallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/phoneverificationchallenge"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (s *Service) VerifyPhone(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	presentedCode string,
	verifiedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (identitydomain.PhoneID, error) {
	if !isSixDigitCode(presentedCode) || now.IsZero() {
		return identitydomain.PhoneID{}, errors.New("verify phone challenge: input is invalid")
	}

	reference, err := phoneVerificationReference(transaction, now)
	if err != nil {
		return identitydomain.PhoneID{}, err
	}
	challengeID := phonechallengedomain.ID(reference.ChallengeID)
	phoneID := identitydomain.PhoneID(reference.ContactID)

	challenge, err := s.phoneChallenges.GetForUpdate(ctx, challengeID)
	if err != nil {
		if errors.Is(err, phonechallengedomain.ErrNotFound) {
			return identitydomain.PhoneID{}, fmt.Errorf("verify phone challenge: %w", ErrChallengeUnavailable)
		}
		return identitydomain.PhoneID{}, fmt.Errorf("verify phone challenge: get challenge: %w", err)
	}
	if challenge.PhoneID != phoneID {
		return identitydomain.PhoneID{}, fmt.Errorf("verify phone challenge: %w", ErrChallengeUnavailable)
	}
	if !s.tokenMatches(presentedCode, string(challenge.TokenHash)) {
		return identitydomain.PhoneID{}, errors.New("verify phone challenge: code does not match")
	}
	if err := challenge.Consume(verifiedFromIP, now); err != nil {
		return identitydomain.PhoneID{}, fmt.Errorf("verify phone challenge: %w: %v", ErrChallengeUnavailable, err)
	}
	if err := s.phoneChallenges.Update(ctx, challenge); err != nil {
		return identitydomain.PhoneID{}, fmt.Errorf("verify phone challenge: save challenge: %w", err)
	}

	return phoneID, nil
}
