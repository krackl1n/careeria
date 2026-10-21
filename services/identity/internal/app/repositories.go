package app

import (
	authenticationrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authenticationflow"
	authorizationcoderepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationcode"
	authorizationgrantrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationgrant"
	authorizationtransactionrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationtransaction"
	emailchallengerepository "github.com/krackl1n/careeria/services/identity/internal/repository/emailverificationchallenge"
	identityrepository "github.com/krackl1n/careeria/services/identity/internal/repository/identity"
	identityprofilerepository "github.com/krackl1n/careeria/services/identity/internal/repository/identityprofile"
	oauthclientrepository "github.com/krackl1n/careeria/services/identity/internal/repository/oauthclient"
	outboxrepository "github.com/krackl1n/careeria/services/identity/internal/repository/outbox"
	passwordrepository "github.com/krackl1n/careeria/services/identity/internal/repository/password"
	phonechallengerepository "github.com/krackl1n/careeria/services/identity/internal/repository/phoneverificationchallenge"
	refreshtokenrepository "github.com/krackl1n/careeria/services/identity/internal/repository/refreshtokenfamily"
	securityeventrepository "github.com/krackl1n/careeria/services/identity/internal/repository/securityevent"
	sessionrepository "github.com/krackl1n/careeria/services/identity/internal/repository/session"
	signingkeyrepository "github.com/krackl1n/careeria/services/identity/internal/repository/signingkey"
)

func (a *App) AuthorizationTransactions() *authorizationtransactionrepository.RedisRepository {
	if a.authorizationTransactions == nil {
		a.authorizationTransactions = authorizationtransactionrepository.NewRedisRepository(a.dependencies.RedisClient)
	}
	return a.authorizationTransactions
}
func (a *App) AuthorizationCodes() *authorizationcoderepository.RedisRepository {
	if a.authorizationCodes == nil {
		a.authorizationCodes = authorizationcoderepository.NewRedisRepository(a.dependencies.RedisClient)
	}
	return a.authorizationCodes
}
func (a *App) AuthorizationGrants() *authorizationgrantrepository.PostgresRepository {
	if a.authorizationGrants == nil {
		a.authorizationGrants = authorizationgrantrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.authorizationGrants
}
func (a *App) Identities() *identityrepository.PostgresRepository {
	if a.identities == nil {
		a.identities = identityrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.identities
}
func (a *App) IdentityProfiles() *identityprofilerepository.PostgresRepository {
	if a.identityProfiles == nil {
		a.identityProfiles = identityprofilerepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.identityProfiles
}
func (a *App) Passwords() *passwordrepository.PostgresRepository {
	if a.passwords == nil {
		a.passwords = passwordrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.passwords
}
func (a *App) OAuthClients() *oauthclientrepository.PostgresRepository {
	if a.oauthClients == nil {
		a.oauthClients = oauthclientrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.oauthClients
}
func (a *App) EmailChallenges() *emailchallengerepository.PostgresRepository {
	if a.emailChallenges == nil {
		a.emailChallenges = emailchallengerepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.emailChallenges
}
func (a *App) PhoneChallenges() *phonechallengerepository.PostgresRepository {
	if a.phoneChallenges == nil {
		a.phoneChallenges = phonechallengerepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.phoneChallenges
}
func (a *App) Sessions() *sessionrepository.PostgresRepository {
	if a.sessions == nil {
		a.sessions = sessionrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.sessions
}
func (a *App) RefreshTokens() *refreshtokenrepository.PostgresRepository {
	if a.refreshTokens == nil {
		a.refreshTokens = refreshtokenrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.refreshTokens
}
func (a *App) SecurityEvents() *securityeventrepository.PostgresRepository {
	if a.securityEvents == nil {
		a.securityEvents = securityeventrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.securityEvents
}
func (a *App) SigningKeys() *signingkeyrepository.PostgresRepository {
	if a.signingKeys == nil {
		a.signingKeys = signingkeyrepository.NewPostgresRepository(a.dependencies.PostgresClient)
	}
	return a.signingKeys
}
func (a *App) Outbox() *outboxrepository.PostgresRepository {
	if a.outbox == nil {
		a.outbox = outboxrepository.NewPostgresRepository(a.dependencies.PostgresClient, 0)
	}
	return a.outbox
}
func (a *App) AuthenticationFacts() *authenticationrepository.PostgresReader {
	if a.authenticationFacts == nil {
		a.authenticationFacts = authenticationrepository.NewPostgresReader(a.dependencies.PostgresClient)
	}
	return a.authenticationFacts
}
