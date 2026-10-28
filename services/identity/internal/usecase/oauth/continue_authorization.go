package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	codedomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

type ContinueAuthorizationInput struct {
	AuthorizationTransactionID uuid.UUID
}

type ContinueAuthorizationOutput struct {
	RedirectURI string
}

func (s *Service) ContinueAuthorization(
	ctx context.Context,
	input ContinueAuthorizationInput,
) (ContinueAuthorizationOutput, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return ContinueAuthorizationOutput{}, apperr.New(apperr.TypeInvalidArgument, "oauth_continue_authorization_invalid_input", "authorization transaction identifier is required", nil)
	}

	now := s.clock.Now()
	transactionID := transactiondomain.ID(input.AuthorizationTransactionID)
	var output ContinueAuthorizationOutput
	err := s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}
		if err := transaction.Complete(now); err != nil {
			return apperr.New(apperr.TypeConflict, "authorization_transaction_not_ready", "authorization transaction is not ready", err)
		}

		code, hash, err := s.tokenGenerator.New()
		if err != nil {
			return fmt.Errorf("generate authorization code: %w", err)
		}
		authorizationCode, err := codedomain.New(
			codedomain.Hash(hash),
			transaction.OAuthClientID,
			transaction.RedirectURI,
			*transaction.IdentityID,
			*transaction.AuthenticationLevel,
			transaction.AuthenticatedMethods,
			transaction.GrantedScopes,
			transaction.Nonce,
			transaction.PKCE,
			*transaction.AuthenticatedAt,
			now.Add(s.cfg.CodeLifetime),
			now,
		)
		if err != nil {
			return fmt.Errorf("create authorization code: %w", err)
		}
		if err := s.authorizationCode.Create(txCtx, authorizationCode); err != nil {
			if errors.Is(err, codedomain.ErrAlreadyExists) {
				return apperr.New(apperr.TypeInternal, "oauth_authorization_code_collision", "cannot issue authorization code", err)
			}
			return fmt.Errorf("store authorization code: %w", err)
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction: %w", err)
		}

		redirectURI, err := authorizedRedirect(transaction, code)
		if err != nil {
			return fmt.Errorf("build authorization redirect: %w", err)
		}
		output = ContinueAuthorizationOutput{RedirectURI: redirectURI}
		return nil
	})
	if err != nil {
		return ContinueAuthorizationOutput{}, err
	}

	return output, nil
}

func authorizedRedirect(transaction *transactiondomain.AuthorizationTransaction, code string) (string, error) {
	redirectURI, err := url.Parse(string(transaction.RedirectURI))
	if err != nil {
		return "", err
	}
	query := redirectURI.Query()
	query.Set("code", code)
	if transaction.State != nil {
		query.Set("state", string(*transaction.State))
	}
	redirectURI.RawQuery = query.Encode()
	return redirectURI.String(), nil
}
