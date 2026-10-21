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

func (s *Service) StartPhone(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	phone identitydomain.Phone,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) error {
	if transaction == nil || !phone.IsActive() {
		return errors.New("start phone verification: input is invalid")
	}

	issued, err := s.issuePhone(ctx, phone.ID, phone.Number, requestedFromIP, now)
	if err != nil {
		return fmt.Errorf("start phone verification: %w", err)
	}
	if err := transaction.BindPhoneVerification(phone.ID, issued.ChallengeID, issued.ExpiresAt, now); err != nil {
		return fmt.Errorf("start phone verification: bind challenge: %w", err)
	}

	return nil
}

func (s *Service) issuePhone(
	ctx context.Context,
	phoneID identitydomain.PhoneID,
	phone identitydomain.PhoneNumber,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (issuedChallenge, error) {
	if !phoneID.IsValid() || phone == "" {
		return issuedChallenge{}, errors.New("issue phone verification challenge: phone is invalid")
	}
	if now.IsZero() {
		return issuedChallenge{}, errors.New("issue phone verification challenge: time is invalid")
	}

	code, codeHash, err := s.generateCode()
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: %w", err)
	}
	expiresAt := now.Add(s.cfg.Lifetime)
	typedTokenHash, err := phonechallengedomain.NewTokenHash(codeHash)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: create token hash: %w", err)
	}
	challenge, err := phonechallengedomain.New(
		phonechallengedomain.ID(s.ids.New()),
		phoneID,
		typedTokenHash,
		s.cfg.MaxAttempts,
		expiresAt,
		requestedFromIP,
		now,
	)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: %w", err)
	}

	if _, err := s.phoneChallenges.RevokeActiveByPhoneID(ctx, phoneID, now); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: revoke active challenge: %w", err)
	}
	if err := s.phoneChallenges.Create(ctx, challenge); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: save challenge: %w", err)
	}

	issuedEvent, err := s.events.Create(
		phonechallengedomain.EventTypeIssued,
		phonechallengedomain.AggregateType,
		challenge.ID.UUID(),
		nil,
		phonechallengedomain.IssuedEvent{
			ChallengeID: challenge.ID.UUID(),
			PhoneID:     challenge.PhoneID.UUID(),
			Phone:       phone,
			Code:        code,
			IssuedAt:    challenge.CreatedAt,
			ExpiresAt:   challenge.ExpiresAt,
		},
		now,
	)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: create issued event: %w", err)
	}
	if err := s.outbox.Create(ctx, issuedEvent); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue phone verification challenge: save issued event: %w", err)
	}

	return issuedChallenge{
		ChallengeID: challenge.ID.UUID(),
		ExpiresAt:   challenge.ExpiresAt,
	}, nil
}
