package authentication

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

type DenyConsentInput struct {
	AuthorizationTransactionID uuid.UUID
}

type DenyConsentOutput struct {
	RedirectURI string
}

func (s *Service) DenyConsent(ctx context.Context, input DenyConsentInput) (DenyConsentOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return DenyConsentOutput{}, apperr.New(apperr.TypeInvalidArgument, "authentication_invalid_input", "authorization transaction identifier is required", nil)
	}

	now := s.clock.Now()
	transactionID := transactiondomain.ID(input.AuthorizationTransactionID)
	var output DenyConsentOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		if err := transaction.Deny(now); err != nil {
			return apperr.New(apperr.TypeConflict, "authorization_consent_invalid_state", "authorization consent cannot be denied", err)
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction: %w", err)
		}

		redirectURI, err := deniedAuthorizationRedirect(transaction)
		if err != nil {
			return fmt.Errorf("build denied authorization redirect: %w", err)
		}
		output = DenyConsentOutput{RedirectURI: redirectURI}
		return nil
	})
	if err != nil {
		return DenyConsentOutput{}, err
	}

	return output, nil
}

func deniedAuthorizationRedirect(transaction *transactiondomain.AuthorizationTransaction) (string, error) {
	redirectURI, err := url.Parse(string(transaction.RedirectURI))
	if err != nil {
		return "", err
	}
	query := redirectURI.Query()
	query.Set("error", "access_denied")
	if transaction.State != nil {
		query.Set("state", string(*transaction.State))
	}
	redirectURI.RawQuery = query.Encode()
	return redirectURI.String(), nil
}
