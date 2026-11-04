package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	codedomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
)

type OAuthClientRepository interface {
	GetByClientID(context.Context, oauthclientdomain.ClientID) (*oauthclientdomain.OAuthClient, error)
	Update(context.Context, *oauthclientdomain.OAuthClient) error
}

type AuthorizationTransactionRepository interface {
	Create(context.Context, *transactiondomain.AuthorizationTransaction) error
	Update(context.Context, *transactiondomain.AuthorizationTransaction) error
	Get(context.Context, transactiondomain.ID) (*transactiondomain.AuthorizationTransaction, error)
}

type AuthenticationRedirectBuilder interface {
	BuildAuthorizationRedirect(transactiondomain.ID) (string, error)
}

type TokenGenerator interface {
	New() (string, string, error)
	Hash(string) string
}

type AuthorizationCodeRepository interface {
	Create(context.Context, *codedomain.AuthorizationCode) error
	Consume(context.Context, codedomain.Hash) (*codedomain.AuthorizationCode, error)
}

type ClientSecretVerifier interface {
	Verify(string, string) (bool, error)
}

type SessionRepository interface {
	Create(context.Context, *sessiondomain.Session) error
	Get(context.Context, sessiondomain.ID) (*sessiondomain.Session, error)
	Update(context.Context, *sessiondomain.Session) error
}

type RefreshTokenFamilyRepository interface {
	Create(context.Context, *refreshtokendomain.RefreshTokenFamily) error
	GetByTokenHashForUpdate(context.Context, refreshtokendomain.TokenHash) (*refreshtokendomain.RefreshTokenFamily, error)
	Update(context.Context, *refreshtokendomain.RefreshTokenFamily) error
}

type AccessTokenIssuer interface {
	Issue(context.Context, accesstokenservice.IssueInput) (accesstokenservice.IssuedToken, error)
}

type AccessTokenVerifier interface {
	Verify(context.Context, string) (accesstokenservice.VerifiedToken, error)
}

type IDTokenIssuer interface {
	Issue(context.Context, idtokenservice.IssueInput) (idtokenservice.IssuedToken, error)
}

type IdentityRepository interface {
	Get(context.Context, identitydomain.ID) (*identitydomain.Identity, error)
}

type IdentityProfileRepository interface {
	Get(context.Context, identitydomain.ID) (*identityprofiledomain.Profile, error)
}

type OutboxWriter interface {
	Create(context.Context, eventdomain.Message) error
}

type SecurityEventWriter interface {
	Create(context.Context, *securityeventdomain.SecurityEvent) error
}

type TokenDependencies struct {
	ClientSecrets        ClientSecretVerifier
	Sessions             SessionRepository
	RefreshTokenFamilies RefreshTokenFamilyRepository
	AccessTokens         AccessTokenIssuer
	AccessTokenVerifier  AccessTokenVerifier
	IDTokens             IDTokenIssuer
	Identities           IdentityRepository
	IdentityProfiles     IdentityProfileRepository
}

type TransactionManager interface {
	Do(context.Context, func(context.Context) error) error
}

type IDGenerator interface {
	New() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	cfg                       Config
	oauthClients              OAuthClientRepository
	authorizationTransactions AuthorizationTransactionRepository
	authenticationRedirects   AuthenticationRedirectBuilder
	tokenGenerator            TokenGenerator
	authorizationCode         AuthorizationCodeRepository
	clientSecrets             ClientSecretVerifier
	sessions                  SessionRepository
	refreshTokenFamilies      RefreshTokenFamilyRepository
	accessTokens              AccessTokenIssuer
	accessTokenVerifier       AccessTokenVerifier
	idTokens                  IDTokenIssuer
	identities                IdentityRepository
	identityProfiles          IdentityProfileRepository
	outbox                    OutboxWriter
	securityEvents            SecurityEventWriter
	events                    eventdomain.Factory
	tx                        TransactionManager
	ids                       IDGenerator
	clock                     Clock
}

func NewService(
	cfg Config,
	oauthClients OAuthClientRepository,
	authorizationTransactions AuthorizationTransactionRepository,
	authenticationRedirects AuthenticationRedirectBuilder,
	tokenGenerator TokenGenerator,
	authorizationCode AuthorizationCodeRepository,
	tokens TokenDependencies,
	outbox OutboxWriter,
	securityEvents SecurityEventWriter,
	tx TransactionManager,
	ids IDGenerator,
	clock Clock,
) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &Service{
		cfg:                       cfg,
		oauthClients:              oauthClients,
		authorizationTransactions: authorizationTransactions,
		authenticationRedirects:   authenticationRedirects,
		tokenGenerator:            tokenGenerator,
		authorizationCode:         authorizationCode,
		clientSecrets:             tokens.ClientSecrets,
		sessions:                  tokens.Sessions,
		refreshTokenFamilies:      tokens.RefreshTokenFamilies,
		accessTokens:              tokens.AccessTokens,
		accessTokenVerifier:       tokens.AccessTokenVerifier,
		idTokens:                  tokens.IDTokens,
		identities:                tokens.Identities,
		identityProfiles:          tokens.IdentityProfiles,
		outbox:                    outbox,
		securityEvents:            securityEvents,
		events:                    eventdomain.NewFactory(ids),
		tx:                        tx,
		ids:                       ids,
		clock:                     clock,
	}, nil
}
