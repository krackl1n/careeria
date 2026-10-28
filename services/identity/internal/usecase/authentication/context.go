package authentication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type GetContextInput struct {
	AuthorizationTransactionID uuid.UUID
}

type ContextOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
	OAuthClient              *oauthclientdomain.OAuthClient
	Identity                 *identitydomain.Identity
	IdentityProfile          *identityprofiledomain.Profile
	AvailableMFAMethods      []authenticationdomain.Method
}

func (s *Service) GetContext(ctx context.Context, input GetContextInput) (ContextOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return ContextOutput{}, apperr.New(apperr.TypeInvalidArgument, "authentication_invalid_input", "authorization transaction identifier is required", nil)
	}

	transactionID := transactiondomain.ID(input.AuthorizationTransactionID)
	var output ContextOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		output, err = s.loadContext(txCtx, transaction)
		return err
	})
	if err != nil {
		return ContextOutput{}, err
	}

	return output, nil
}

func (s *Service) LoadContext(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
) (ContextOutput, error) {
	if transaction == nil {
		return ContextOutput{}, errors.New("load authentication context: authorization transaction is missing")
	}

	var output ContextOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		var err error
		output, err = s.loadContext(txCtx, transaction)
		return err
	})
	if err != nil {
		return ContextOutput{}, err
	}
	return output, nil
}

func (s *Service) loadContext(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
) (ContextOutput, error) {
	client, err := s.oauthClients.Get(ctx, transaction.OAuthClientID)
	if err != nil {
		if errors.Is(err, oauthclientdomain.ErrNotFound) {
			return ContextOutput{}, apperr.New(apperr.TypeNotFound, "oauth_client_not_found", "OAuth client not found", err)
		}
		return ContextOutput{}, fmt.Errorf("get OAuth client: %w", err)
	}

	output := ContextOutput{AuthorizationTransaction: transaction, OAuthClient: client}
	if transaction.IdentityID == nil {
		return output, nil
	}

	identity, err := s.identities.Get(ctx, *transaction.IdentityID)
	if err != nil {
		return ContextOutput{}, fmt.Errorf("get authorization transaction identity: %w", err)
	}
	profile, err := s.identityProfiles.Get(ctx, *transaction.IdentityID)
	if err != nil {
		return ContextOutput{}, fmt.Errorf("get authorization transaction identity profile: %w", err)
	}
	output.Identity = identity
	output.IdentityProfile = profile
	if transaction.NextStep == transactiondomain.NextStepMFA {
		facts, err := s.authenticationFlowFacts.Get(
			ctx,
			identity.ID,
			transaction.OAuthClientID,
			transaction.RequestedScopes,
			s.clock.Now(),
		)
		if err != nil {
			return ContextOutput{}, fmt.Errorf("get authentication context MFA methods: %w", err)
		}
		output.AvailableMFAMethods = facts.AvailableMFAMethods()
	}
	return output, nil
}
