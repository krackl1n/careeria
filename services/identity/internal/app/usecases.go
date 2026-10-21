package app

import (
	"fmt"
	"time"

	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	jwtsigner "github.com/krackl1n/careeria/services/identity/internal/signer/jwt"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
	metadatausecase "github.com/krackl1n/careeria/services/identity/internal/usecase/metadata"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	oauthclientadminusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauthclientadmin"
	signingkeyadminusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/signingkeyadmin"
	jwtverifier "github.com/krackl1n/careeria/services/identity/internal/verifier/jwt"
	"github.com/krackl1n/careeria/services/identity/pkg/password"
	"github.com/krackl1n/careeria/services/identity/pkg/token"
)

func (a *App) SigningKeyAdmin() (*signingkeyadminusecase.Service, error) {
	if a.signingKeyAdmin != nil {
		return a.signingKeyAdmin, nil
	}
	signer, err := a.Signer()
	if err != nil {
		return nil, fmt.Errorf("initialize signer: %w", err)
	}
	material := signer.KeyMaterial()
	a.signingKeyAdmin = signingkeyadminusecase.NewService(
		a.SigningKeys(),
		a.IDGenerator(),
		a.Clock(),
		signingkeyadminusecase.KeyMaterial{
			KeyID:         material.KeyID,
			Algorithm:     material.Algorithm,
			PublicJWK:     material.PublicJWK,
			PrivateKeyRef: material.PrivateKeyRef,
		},
	)
	return a.signingKeyAdmin, nil
}

func (a *App) OAuthClientAdmin() *oauthclientadminusecase.Service {
	if a.oauthClientAdmin == nil {
		a.oauthClientAdmin = oauthclientadminusecase.NewService(
			a.OAuthClients(),
			a.TransactionManager(),
			a.IDGenerator(),
			a.Clock(),
		)
	}
	return a.oauthClientAdmin
}

func (a *App) Signer() (*jwtsigner.Signer, error) {
	if a.signer != nil {
		return a.signer, nil
	}
	signer, err := jwtsigner.NewSigner(a.dependencies.Config.JWTSigner)
	if err != nil {
		return nil, err
	}
	a.signer = signer
	return signer, nil
}
func (a *App) AccessTokens() (*accesstokenservice.Service, error) {
	if a.accessTokens != nil {
		return a.accessTokens, nil
	}
	signer, err := a.Signer()
	if err != nil {
		return nil, err
	}
	service, err := accesstokenservice.NewService(a.dependencies.Config.AccessToken, a.SigningKeys(), signer, jwtverifier.Verifier{}, a.IDGenerator(), a.Clock())
	if err != nil {
		return nil, err
	}
	a.accessTokens = service
	return service, nil
}
func (a *App) IDTokens() (*idtokenservice.Service, error) {
	if a.idTokens != nil {
		return a.idTokens, nil
	}
	signer, err := a.Signer()
	if err != nil {
		return nil, err
	}
	service, err := idtokenservice.NewService(a.dependencies.Config.IDToken, a.SigningKeys(), signer, a.Clock())
	if err != nil {
		return nil, err
	}
	a.idTokens = service
	return service, nil
}
func (a *App) ContactVerification() (*contactverificationservice.Service, error) {
	if a.contactVerification != nil {
		return a.contactVerification, nil
	}
	service, err := contactverificationservice.NewService(a.dependencies.Config.ContactVerification, a.EmailChallenges(), a.PhoneChallenges(), a.Outbox(), token.NewSixDigitCode(), token.NewSHA256(), a.IDGenerator())
	if err != nil {
		return nil, err
	}
	a.contactVerification = service
	return service, nil
}
func (a *App) Authentication() (*authenticationusecase.Service, error) {
	if a.authentication != nil {
		return a.authentication, nil
	}
	contacts, err := a.ContactVerification()
	if err != nil {
		return nil, err
	}
	hasher, err := password.NewArgon2id(password.DefaultParams())
	if err != nil {
		return nil, err
	}
	a.authentication = authenticationusecase.NewService(a.AuthorizationTransactions(), a.OAuthClients(), a.AuthorizationGrants(), a.Identities(), a.IdentityProfiles(), a.Outbox(), eventdomain.NewFactory(a.IDGenerator()), a.Passwords(), a.AuthenticationFacts(), contacts, passwordHasher{hasher}, passwordVerifier{hasher}, passworddomain.FailurePolicy{MaxAttempts: 5, LockDuration: 15 * time.Minute}, a.TransactionManager(), a.IDGenerator(), a.Clock())
	return a.authentication, nil
}
func (a *App) OAuth() (*oauthusecase.Service, error) {
	if a.oauth != nil {
		return a.oauth, nil
	}
	generator, err := token.NewRandom(token.DefaultParams())
	if err != nil {
		return nil, err
	}
	accessTokens, err := a.AccessTokens()
	if err != nil {
		return nil, err
	}
	idTokens, err := a.IDTokens()
	if err != nil {
		return nil, err
	}
	hasher, err := password.NewArgon2id(password.DefaultParams())
	if err != nil {
		return nil, err
	}
	service, err := oauthusecase.NewService(a.dependencies.Config.OAuth, a.OAuthClients(), a.AuthorizationTransactions(), authenticationRedirect{baseURL: a.dependencies.Config.AuthenticationRedirectURI}, tokenGenerator{generator: generator, digester: token.NewSHA256()}, a.AuthorizationCodes(), oauthusecase.TokenDependencies{ClientSecrets: clientSecretVerifier{hasher}, Sessions: a.Sessions(), RefreshTokenFamilies: a.RefreshTokens(), AccessTokens: accessTokens, AccessTokenVerifier: accessTokens, IDTokens: idTokens, Identities: a.Identities(), IdentityProfiles: a.IdentityProfiles()}, a.Outbox(), a.SecurityEvents(), a.TransactionManager(), a.IDGenerator(), a.Clock())
	if err != nil {
		return nil, fmt.Errorf("create OAuth service: %w", err)
	}
	a.oauth = service
	return service, nil
}
func (a *App) Metadata() (*metadatausecase.Service, error) {
	if a.metadata != nil {
		return a.metadata, nil
	}
	service, err := metadatausecase.NewService(metadatausecase.Config{Issuer: a.dependencies.Config.IDToken.Issuer, IDTokenSigningAlgorithm: a.dependencies.Config.IDToken.Algorithm}, a.SigningKeys(), a.Clock())
	if err != nil {
		return nil, err
	}
	a.metadata = service
	return service, nil
}
