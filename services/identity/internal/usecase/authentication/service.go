package authentication

import (
	"context"
	"time"

	"github.com/google/uuid"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	grantdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationgrant"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type AuthorizationTransactionRepository interface {
	Get(context.Context, transactiondomain.ID) (*transactiondomain.AuthorizationTransaction, error)
	Update(context.Context, *transactiondomain.AuthorizationTransaction) error
}

type OAuthClientRepository interface {
	Get(context.Context, oauthclientdomain.ID) (*oauthclientdomain.OAuthClient, error)
}

type AuthorizationGrantRepository interface {
	Get(context.Context, identitydomain.ID, oauthclientdomain.ID) (*grantdomain.AuthorizationGrant, error)
	Create(context.Context, *grantdomain.AuthorizationGrant) error
	Update(context.Context, *grantdomain.AuthorizationGrant) error
}

type IdentityRepository interface {
	Create(context.Context, *identitydomain.Identity) error
	Get(context.Context, identitydomain.ID) (*identitydomain.Identity, error)
	GetByEmail(context.Context, identitydomain.EmailAddress) (*identitydomain.Identity, error)
	GetByPhone(context.Context, identitydomain.PhoneNumber) (*identitydomain.Identity, error)
	Update(context.Context, *identitydomain.Identity) error
}

type IdentityProfileRepository interface {
	Create(context.Context, *identityprofiledomain.Profile) error
	Get(context.Context, identitydomain.ID) (*identityprofiledomain.Profile, error)
}

type OutboxWriter interface {
	Create(context.Context, eventdomain.Message) error
}

type PasswordRepository interface {
	Create(context.Context, *passworddomain.Password) error
	Get(context.Context, identitydomain.ID) (*passworddomain.Password, error)
	Update(context.Context, *passworddomain.Password) error
}

type AuthenticationFlowFactsReader interface {
	Get(
		context.Context,
		identitydomain.ID,
		oauthclientdomain.ID,
		[]oauthdomain.Scope,
		time.Time,
	) (flowdomain.Facts, error)
}

type ContactVerificationService interface {
	StartEmail(
		context.Context,
		*transactiondomain.AuthorizationTransaction,
		identitydomain.Email,
		*sessiondomain.IPAddress,
		time.Time,
	) error
	StartPhone(
		context.Context,
		*transactiondomain.AuthorizationTransaction,
		identitydomain.Phone,
		*sessiondomain.IPAddress,
		time.Time,
	) error
	VerifyEmail(
		context.Context,
		*transactiondomain.AuthorizationTransaction,
		string,
		*sessiondomain.IPAddress,
		time.Time,
	) (identitydomain.EmailID, error)
	VerifyPhone(
		context.Context,
		*transactiondomain.AuthorizationTransaction,
		string,
		*sessiondomain.IPAddress,
		time.Time,
	) (identitydomain.PhoneID, error)
}

type PasswordVerifier interface {
	Verify(password string, hash passworddomain.Hash) error
}

type PasswordHasher interface {
	Hash(string) (string, error)
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
	authorizationTransactions AuthorizationTransactionRepository
	oauthClients              OAuthClientRepository
	authorizationGrants       AuthorizationGrantRepository
	identities                IdentityRepository
	identityProfiles          IdentityProfileRepository
	outbox                    OutboxWriter
	events                    eventdomain.Factory
	passwords                 PasswordRepository
	authenticationFlowFacts   AuthenticationFlowFactsReader
	contactVerifications      ContactVerificationService
	passwordHasher            PasswordHasher
	passwordVerifier          PasswordVerifier
	passwordFailurePolicy     passworddomain.FailurePolicy
	tx                        TransactionManager
	ids                       IDGenerator
	clock                     Clock
}

func NewService(
	authorizationTransactions AuthorizationTransactionRepository,
	oauthClients OAuthClientRepository,
	authorizationGrants AuthorizationGrantRepository,
	identities IdentityRepository,
	identityProfiles IdentityProfileRepository,
	outbox OutboxWriter,
	events eventdomain.Factory,
	passwords PasswordRepository,
	authenticationFlowFacts AuthenticationFlowFactsReader,
	contactVerifications ContactVerificationService,
	passwordHasher PasswordHasher,
	passwordVerifier PasswordVerifier,
	passwordFailurePolicy passworddomain.FailurePolicy,
	tx TransactionManager,
	ids IDGenerator,
	clock Clock,
) *Service {
	return &Service{
		authorizationTransactions: authorizationTransactions,
		oauthClients:              oauthClients,
		authorizationGrants:       authorizationGrants,
		identities:                identities,
		identityProfiles:          identityProfiles,
		outbox:                    outbox,
		events:                    events,
		passwords:                 passwords,
		authenticationFlowFacts:   authenticationFlowFacts,
		contactVerifications:      contactVerifications,
		passwordHasher:            passwordHasher,
		passwordVerifier:          passwordVerifier,
		passwordFailurePolicy:     passwordFailurePolicy,
		tx:                        tx,
		ids:                       ids,
		clock:                     clock,
	}
}
