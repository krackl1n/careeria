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

func (s *Service) StartEmail(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	email identitydomain.Email,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) error {
	if transaction == nil || !email.IsActive() {
		return errors.New("start email verification: input is invalid")
	}

	issued, err := s.issueEmail(ctx, email.ID, email.Address, requestedFromIP, now)
	if err != nil {
		return fmt.Errorf("start email verification: %w", err)
	}
	if err := transaction.BindEmailVerification(email.ID, issued.ChallengeID, issued.ExpiresAt, now); err != nil {
		return fmt.Errorf("start email verification: bind challenge: %w", err)
	}

	return nil
}

func (s *Service) issueEmail(
	ctx context.Context,
	emailID identitydomain.EmailID,
	email identitydomain.EmailAddress,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (issuedChallenge, error) {
	if !emailID.IsValid() || email == "" {
		return issuedChallenge{}, errors.New("issue email verification challenge: email is invalid")
	}
	if now.IsZero() {
		return issuedChallenge{}, errors.New("issue email verification challenge: time is invalid")
	}

	code, codeHash, err := s.generateCode()
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: %w", err)
	}
	expiresAt := now.Add(s.cfg.Lifetime)
	typedTokenHash, err := emailchallengedomain.NewTokenHash(codeHash)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: create token hash: %w", err)
	}

	challenge, err := emailchallengedomain.New(
		emailchallengedomain.ID(s.ids.New()),
		emailID,
		typedTokenHash,
		s.cfg.MaxAttempts,
		expiresAt,
		requestedFromIP,
		now,
	)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: %w", err)
	}

	if _, err := s.emailChallenges.RevokeActiveByEmailID(ctx, emailID, now); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: revoke active challenge: %w", err)
	}
	if err := s.emailChallenges.Create(ctx, challenge); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: save challenge: %w", err)
	}

	issuedEvent, err := s.events.Create(
		emailchallengedomain.EventTypeIssued,
		emailchallengedomain.AggregateType,
		challenge.ID.UUID(),
		nil,
		emailchallengedomain.IssuedEvent{
			ChallengeID: challenge.ID.UUID(),
			EmailID:     challenge.EmailID.UUID(),
			Email:       email,
			Code:        code,
			IssuedAt:    challenge.CreatedAt,
			ExpiresAt:   challenge.ExpiresAt,
		},
		now,
	)
	if err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: create issued event: %w", err)
	}
	if err := s.outbox.Create(ctx, issuedEvent); err != nil {
		return issuedChallenge{}, fmt.Errorf("issue email verification challenge: save issued event: %w", err)
	}

	return issuedChallenge{
		ChallengeID: challenge.ID.UUID(),
		ExpiresAt:   challenge.ExpiresAt,
	}, nil
}
