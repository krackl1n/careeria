package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	grantdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationgrant"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
)

type GrantConsentInput struct {
	AuthorizationTransactionID uuid.UUID
	GrantedScopes              []string
	Remember                   bool
}

type GrantConsentOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

type cmdGrantConsent struct {
	transactionID transactiondomain.ID
	grantedScopes []oauthdomain.Scope
	remember      bool
}

func (s *Service) GrantConsent(ctx context.Context, input GrantConsentInput) (GrantConsentOutput, error) {
	command, err := prepareGrantConsent(input)
	if err != nil {
		return GrantConsentOutput{}, apperr.New(apperr.TypeInvalidArgument, "authorization_consent_invalid_input", "authorization consent is invalid", err)
	}

	now := s.clock.Now()
	var output GrantConsentOutput
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, command.transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		if err := transaction.AuthorizeScopes(command.grantedScopes, transactiondomain.AuthorizationSourceUserConsent, now); err != nil {
			return apperr.New(apperr.TypeConflict, "authorization_consent_invalid_state", "authorization consent cannot be granted", err)
		}

		if command.remember {
			if err := s.rememberConsent(txCtx, transaction, command.grantedScopes, now); err != nil {
				return err
			}
		}

		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction: %w", err)
		}
		output = GrantConsentOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return GrantConsentOutput{}, err
	}

	return output, nil
}

func prepareGrantConsent(input GrantConsentInput) (cmdGrantConsent, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return cmdGrantConsent{}, errors.New("authorization transaction identifier is required")
	}
	if len(input.GrantedScopes) == 0 {
		return cmdGrantConsent{}, errors.New("at least one granted scope is required")
	}

	scopes := make([]oauthdomain.Scope, 0, len(input.GrantedScopes))
	seen := make(map[oauthdomain.Scope]struct{}, len(input.GrantedScopes))
	for _, rawScope := range input.GrantedScopes {
		scope, err := oauthdomain.NewScope(rawScope)
		if err != nil {
			return cmdGrantConsent{}, fmt.Errorf("create OAuth scope: %w", err)
		}
		if _, exists := seen[scope]; exists {
			return cmdGrantConsent{}, errors.New("OAuth scope is duplicated")
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	return cmdGrantConsent{
		transactionID: transactiondomain.ID(input.AuthorizationTransactionID),
		grantedScopes: scopes,
		remember:      input.Remember,
	}, nil
}

func (s *Service) rememberConsent(
	ctx context.Context,
	transaction *transactiondomain.AuthorizationTransaction,
	scopes []oauthdomain.Scope,
	now time.Time,
) error {
	if transaction.IdentityID == nil {
		return apperr.New(apperr.TypeConflict, "authorization_consent_invalid_state", "authorization consent cannot be granted", errors.New("authorization transaction identity is missing"))
	}

	grant, err := s.authorizationGrants.Get(ctx, *transaction.IdentityID, transaction.OAuthClientID)
	if err != nil && !errors.Is(err, grantdomain.ErrNotFound) {
		return fmt.Errorf("get authorization grant: %w", err)
	}
	if errors.Is(err, grantdomain.ErrNotFound) {
		grant, err = grantdomain.New(
			grantdomain.ID(s.ids.New()),
			*transaction.IdentityID,
			transaction.OAuthClientID,
			grantdomain.SourceUserConsent,
			scopes,
			nil,
			now,
		)
		if err != nil {
			return fmt.Errorf("create authorization grant: %w", err)
		}
		if err := s.authorizationGrants.Create(ctx, grant); err != nil {
			return fmt.Errorf("create authorization grant: %w", err)
		}
		return nil
	}

	if err := grant.Replace(grantdomain.SourceUserConsent, scopes, nil, now); err != nil {
		return fmt.Errorf("replace authorization grant: %w", err)
	}
	if err := s.authorizationGrants.Update(ctx, grant); err != nil {
		return fmt.Errorf("update authorization grant: %w", err)
	}
	return nil
}
