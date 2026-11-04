package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"

	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (s *Service) resolveNextStep(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	identityID identitydomain.ID,
	requirements flowdomain.Requirements,
	now time.Time,
) (flowdomain.Decision, error) {
	if transaction == nil || !identityID.IsValid() {
		return flowdomain.Decision{}, errors.New("authentication flow state is incomplete")
	}

	input := flowdomain.Input{
		CurrentStep:  transaction.NextStep,
		Requirements: requirements,
	}
	decision, err := flowdomain.ResolveNextStep(input)
	if !errors.Is(err, flowdomain.ErrFactsRequired) {
		return decision, err
	}

	facts, err := s.authenticationFlowFacts.Get(
		ctx,
		identityID,
		transaction.OAuthClientID,
		transaction.RequestedScopes,
		now,
	)
	if err != nil {
		return flowdomain.Decision{}, fmt.Errorf("get authentication flow facts: %w", err)
	}
	input.Facts = &facts
	return flowdomain.ResolveNextStep(input)
}
