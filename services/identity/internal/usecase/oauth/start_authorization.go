package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

const (
	responseTypeCode = "code"
	openIDScope      = oauthdomain.Scope("openid")
)

type StartAuthorizationInput struct {
	ResponseType string
	ClientID     string
	RedirectURI  string
	Scope        string

	State *string
	Nonce *string

	CodeChallenge       *string
	CodeChallengeMethod *string
}

type StartAuthorizationOutput struct {
	RedirectURI                string
	AuthorizationTransactionID uuid.UUID
	ExpiresAt                  time.Time
}

type cmdStartAuthorization struct {
	clientID    oauthclientdomain.ClientID
	redirectURI oauthclientdomain.RedirectURIValue
	scopes      []oauthdomain.Scope
	state       *transactiondomain.State
	nonce       *openidconnectdomain.Nonce
	pkce        *oauthdomain.PKCE
}

func (s *Service) StartAuthorization(ctx context.Context, input StartAuthorizationInput) (StartAuthorizationOutput, error) {
	request, err := prepareStartAuthorization(input)
	if err != nil {
		return StartAuthorizationOutput{}, apperr.New(apperr.TypeInvalidArgument, "oauth_authorization_invalid_input", "OAuth authorization request is invalid", err)
	}

	now := s.clock.Now()
	var output StartAuthorizationOutput
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		oauthClient, err := s.oauthClients.GetByClientID(txCtx, request.clientID)
		if err != nil {
			if errors.Is(err, oauthclientdomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "oauth_client_not_found", "OAuth client not found", err)
			}

			return fmt.Errorf("get OAuth client for authorization: %w", err)
		}

		if err := oauthClient.ValidateAuthorizationRedirectURI(request.redirectURI); err != nil {
			if errors.Is(err, oauthclientdomain.ErrRedirectURINotFound) {
				return apperr.New(apperr.TypeInvalidArgument, "oauth_redirect_uri_invalid", "OAuth redirect URI is not registered for the client", err)
			}

			return fmt.Errorf("validate OAuth authorization redirect URI: %w", err)
		}
		if err := oauthClient.ValidateAuthorizationCodeFlow(request.scopes, now); err != nil {
			switch {
			case errors.Is(err, oauthclientdomain.ErrInvalidStatus), errors.Is(err, oauthclientdomain.ErrAuthorizationCodeFlowNotAllowed):
				return apperr.New(apperr.TypeForbidden, "oauth_client_unauthorized", "OAuth client is not allowed to use authorization code flow", err)
			case errors.Is(err, oauthclientdomain.ErrScopeNotFound):
				return apperr.New(apperr.TypeInvalidArgument, "oauth_scope_not_allowed", "OAuth scope is not allowed", err)
			default:
				return fmt.Errorf("validate OAuth authorization code flow: %w", err)
			}
		}

		transaction, err := transactiondomain.New(
			transactiondomain.ID(s.ids.New()),
			oauthClient.ID,
			request.redirectURI,
			request.scopes,
			request.state,
			request.nonce,
			oauthClient.RequirePKCE,
			request.pkce,
			now.Add(s.cfg.AuthorizationLifetime),
			now,
		)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrPKCERequired) {
				return apperr.New(apperr.TypeInvalidArgument, "oauth_pkce_required", "OAuth PKCE parameters are required", err)
			}

			return fmt.Errorf("create authorization transaction entity: %w", err)
		}

		authenticationRedirect, err := s.authenticationRedirects.BuildAuthorizationRedirect(transaction.ID)
		if err != nil {
			return fmt.Errorf("build authentication redirect: %w", err)
		}
		if err := s.authorizationTransactions.Create(txCtx, transaction); err != nil {
			return fmt.Errorf("save authorization transaction: %w", err)
		}

		output = StartAuthorizationOutput{
			RedirectURI:                authenticationRedirect,
			AuthorizationTransactionID: transaction.ID.UUID(),
			ExpiresAt:                  transaction.ExpiresAt,
		}
		return nil
	})
	if err != nil {
		return StartAuthorizationOutput{}, err
	}

	return output, nil
}

func prepareStartAuthorization(input StartAuthorizationInput) (cmdStartAuthorization, error) {
	if input.ResponseType != responseTypeCode {
		return cmdStartAuthorization{}, errors.New("only response_type=code is supported")
	}

	clientID, err := oauthclientdomain.NewClientID(input.ClientID)
	if err != nil {
		return cmdStartAuthorization{}, fmt.Errorf("create OAuth client identifier: %w", err)
	}
	redirectURI, err := oauthclientdomain.NewRedirectURI(input.RedirectURI)
	if err != nil {
		return cmdStartAuthorization{}, fmt.Errorf("create OAuth redirect URI: %w", err)
	}

	if input.Scope == "" {
		return cmdStartAuthorization{}, errors.New("OAuth scope is required")
	}
	scopes := make([]oauthdomain.Scope, 0, strings.Count(input.Scope, " ")+1)
	seen := make(map[oauthdomain.Scope]struct{}, cap(scopes))
	hasOpenID := false
	for _, value := range strings.Split(input.Scope, " ") {
		scope, err := oauthdomain.NewScope(value)
		if err != nil {
			return cmdStartAuthorization{}, fmt.Errorf("create OAuth scope: %w", err)
		}
		if _, exists := seen[scope]; exists {
			return cmdStartAuthorization{}, errors.New("OAuth scope is duplicated")
		}

		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
		hasOpenID = hasOpenID || scope == openIDScope
	}
	if !hasOpenID {
		return cmdStartAuthorization{}, errors.New("OpenID Connect scope openid is required")
	}

	var state *transactiondomain.State
	if input.State != nil {
		value, err := transactiondomain.NewState(*input.State)
		if err != nil {
			return cmdStartAuthorization{}, fmt.Errorf("create OAuth state: %w", err)
		}
		state = &value
	}

	var nonce *openidconnectdomain.Nonce
	if input.Nonce != nil {
		value, err := openidconnectdomain.NewNonce(*input.Nonce)
		if err != nil {
			return cmdStartAuthorization{}, fmt.Errorf("create OpenID Connect nonce: %w", err)
		}
		nonce = &value
	}

	pkce, err := oauthdomain.NewPKCE(input.CodeChallenge, input.CodeChallengeMethod)
	if err != nil {
		return cmdStartAuthorization{}, fmt.Errorf("create OAuth PKCE parameters: %w", err)
	}

	return cmdStartAuthorization{
		clientID:    clientID,
		redirectURI: redirectURI,
		scopes:      scopes,
		state:       state,
		nonce:       nonce,
		pkce:        pkce,
	}, nil
}
