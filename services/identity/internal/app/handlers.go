package app

import (
	authenticationtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/authentication"
	metadatatransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/metadata"
	oauthtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/oauth"
	oauthclientadmintransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/oauthclientadmin"
	registrationtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/registration"
	signingkeyadmintransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/signingkeyadmin"
)

func (a *App) SigningKeyAdminHandler() (*signingkeyadmintransport.Handler, error) {
	if a.signingKeyAdminHandler != nil {
		return a.signingKeyAdminHandler, nil
	}
	service, err := a.SigningKeyAdmin()
	if err != nil {
		return nil, err
	}
	a.signingKeyAdminHandler = signingkeyadmintransport.NewHandler(service)
	return a.signingKeyAdminHandler, nil
}

func (a *App) OAuthClientAdminHandler() *oauthclientadmintransport.Handler {
	if a.oauthClientAdminHandler == nil {
		a.oauthClientAdminHandler = oauthclientadmintransport.NewHandler(a.OAuthClientAdmin())
	}
	return a.oauthClientAdminHandler
}

func (a *App) AuthenticationHandler() (*authenticationtransport.AuthenticationHandler, error) {
	if a.authenticationHandler != nil {
		return a.authenticationHandler, nil
	}
	service, err := a.Authentication()
	if err != nil {
		return nil, err
	}
	a.authenticationHandler = authenticationtransport.NewAuthenticationHandler(service)
	return a.authenticationHandler, nil
}
func (a *App) RegistrationHandler() (*registrationtransport.Handler, error) {
	if a.registrationHandler != nil {
		return a.registrationHandler, nil
	}
	service, err := a.Authentication()
	if err != nil {
		return nil, err
	}
	a.registrationHandler = registrationtransport.NewHandler(service)
	return a.registrationHandler, nil
}
func (a *App) OAuthHandler() (*oauthtransport.Handler, error) {
	if a.oauthHandler != nil {
		return a.oauthHandler, nil
	}
	service, err := a.OAuth()
	if err != nil {
		return nil, err
	}
	a.oauthHandler = oauthtransport.NewHandler(service)
	return a.oauthHandler, nil
}

func (a *App) MetadataHandler() (*metadatatransport.Handler, error) {
	if a.metadataHandler != nil {
		return a.metadataHandler, nil
	}
	service, err := a.Metadata()
	if err != nil {
		return nil, err
	}
	a.metadataHandler = metadatatransport.NewHandler(service)
	return a.metadataHandler, nil
}
