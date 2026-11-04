package authorizationtransaction

import (
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

type AuthorizationTransaction struct {
	ID ID

	OAuthClientID oauthclientdomain.ID

	RedirectURI     oauthclientdomain.RedirectURIValue
	RequestedScopes []oauthdomain.Scope
	State           *State
	Nonce           *openidconnectdomain.Nonce
	PKCE            *oauthdomain.PKCE

	Status   Status
	NextStep NextStep

	IdentityID           *identitydomain.ID
	AuthenticationLevel  *authenticationdomain.Level
	AuthenticatedMethods []authenticationdomain.Method
	GrantedScopes        []oauthdomain.Scope
	AuthorizationSource  *AuthorizationSource
	Verification         *VerificationReference

	AuthenticatedAt *time.Time
	ExpiresAt       time.Time
}

func New(
	id ID,
	oauthClientID oauthclientdomain.ID,
	redirectURI oauthclientdomain.RedirectURIValue,
	requestedScopes []oauthdomain.Scope,
	state *State,
	nonce *openidconnectdomain.Nonce,
	requirePKCE bool,
	pkce *oauthdomain.PKCE,
	expiresAt time.Time,
	now time.Time,
) (*AuthorizationTransaction, error) {
	if requirePKCE && pkce == nil {
		return nil, fmt.Errorf("create authorization transaction: %w", ErrPKCERequired)
	}
	if !id.IsValid() {
		return nil, errors.New("create authorization transaction: identifier is required")
	}
	if !oauthClientID.IsValid() {
		return nil, errors.New("create authorization transaction: OAuth client identifier is invalid")
	}
	if domain.HasDuplicates(requestedScopes) {
		return nil, errors.New("create authorization transaction: requested scope is duplicated")
	}
	if now.IsZero() || !expiresAt.After(now) {
		return nil, errors.New("create authorization transaction: expiration is invalid")
	}

	return &AuthorizationTransaction{
		ID: id,

		OAuthClientID: oauthClientID,

		RedirectURI:     redirectURI,
		RequestedScopes: slices.Clone(requestedScopes),
		State:           domain.ClonePointer(state),
		Nonce:           domain.ClonePointer(nonce),
		PKCE:            domain.ClonePointer(pkce),

		Status:   StatusPending,
		NextStep: NextStepChooseAuthenticationMethod,

		ExpiresAt: expiresAt,
	}, nil
}

func (t *AuthorizationTransaction) AuthorizeScopes(
	grantedScopes []oauthdomain.Scope,
	source AuthorizationSource,
	now time.Time,
) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("authorize transaction scopes: %w", err)
	}
	if t.Status != StatusPending || t.NextStep != NextStepConsent || t.AuthenticationLevel == nil {
		return fmt.Errorf("authorize transaction scopes: %w", ErrInvalidStatus)
	}

	if len(grantedScopes) == 0 {
		return errors.New("authorize transaction scopes: at least one scope is required")
	}
	if err := t.ensureGrantedScopesAllowed(grantedScopes); err != nil {
		return fmt.Errorf("authorize transaction scopes: %w", err)
	}
	if err := t.transitionTo(NextStepReady); err != nil {
		return fmt.Errorf("authorize transaction scopes: %w", err)
	}

	t.GrantedScopes = slices.Clone(grantedScopes)
	t.AuthorizationSource = domain.ClonePointer(&source)
	t.Status = StatusReady
	return nil
}

func (t *AuthorizationTransaction) Complete(now time.Time) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("complete authorization transaction: %w", err)
	}
	if t.Status != StatusReady || t.NextStep != NextStepReady {
		return fmt.Errorf("complete authorization transaction: %w", ErrInvalidStatus)
	}

	if t.IdentityID == nil || t.AuthenticationLevel == nil || t.AuthenticatedAt == nil {
		return fmt.Errorf("complete authorization transaction: completed transaction authentication snapshot is missing")
	}

	t.finish(StatusCompleted)
	return nil
}

func (t *AuthorizationTransaction) Deny(now time.Time) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("deny authorization transaction: %w", err)
	}
	if t.Status != StatusPending {
		return fmt.Errorf("deny authorization transaction: %w", ErrInvalidStatus)
	}

	t.finish(StatusDenied)
	return nil
}

func (t *AuthorizationTransaction) transitionTo(step NextStep) error {
	if t.Status != StatusPending {
		return ErrInvalidStatus
	}
	if !t.NextStep.CanTransitionTo(step) {
		return ErrInvalidStep
	}

	t.NextStep = step
	return nil
}

func (t *AuthorizationTransaction) finish(status Status) {
	t.Status = status
	t.NextStep = ""
	t.Verification = nil
}

func (t *AuthorizationTransaction) ensureChangeAllowed(now time.Time) error {
	if t == nil {
		return errors.New("validate authorization transaction for change: transaction is nil")
	}
	if now.IsZero() {
		return errors.New("validate authorization transaction for change: time is invalid")
	}
	if !now.Before(t.ExpiresAt) {
		return fmt.Errorf("validate authorization transaction for change: %w", ErrExpired)
	}

	return nil
}

func (t *AuthorizationTransaction) ensureGrantedScopesAllowed(granted []oauthdomain.Scope) error {
	if domain.HasDuplicates(granted) {
		return errors.New("granted authorization scope is duplicated")
	}
	requested := make(map[oauthdomain.Scope]struct{}, len(t.RequestedScopes))
	for _, scope := range t.RequestedScopes {
		requested[scope] = struct{}{}
	}
	for _, scope := range granted {
		if _, exists := requested[scope]; !exists {
			return errors.New("granted scope was not requested")
		}
	}

	return nil
}
