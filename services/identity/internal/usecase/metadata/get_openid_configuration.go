package metadata

import "strings"

type OpenIDConfiguration struct {
	Issuer                                 string
	AuthorizationEndpoint                  string
	TokenEndpoint                          string
	RevocationEndpoint                     string
	UserInfoEndpoint                       string
	JSONWebKeySetURI                       string
	ResponseTypesSupported                 []string
	SubjectTypesSupported                  []string
	IDTokenSigningAlgorithmsSupported      []string
	ScopesSupported                        []string
	ClaimsSupported                        []string
	GrantTypesSupported                    []string
	TokenEndpointAuthMethodsSupported      []string
	RevocationEndpointAuthMethodsSupported []string
	CodeChallengeMethodsSupported          []string
}

func (s *Service) GetOpenIDConfiguration() OpenIDConfiguration {
	baseURL := strings.TrimRight(s.cfg.Issuer, "/")

	return OpenIDConfiguration{
		Issuer:                                 s.cfg.Issuer,
		AuthorizationEndpoint:                  baseURL + "/oauth2/authorize",
		TokenEndpoint:                          baseURL + "/oauth2/token",
		RevocationEndpoint:                     baseURL + "/oauth2/revoke",
		UserInfoEndpoint:                       baseURL + "/oauth2/userinfo",
		JSONWebKeySetURI:                       baseURL + "/.well-known/jwks.json",
		ResponseTypesSupported:                 []string{"code"},
		SubjectTypesSupported:                  []string{"public"},
		IDTokenSigningAlgorithmsSupported:      []string{s.cfg.IDTokenSigningAlgorithm},
		ScopesSupported:                        []string{"openid", "profile", "email", "phone"},
		ClaimsSupported:                        []string{"sub", "name", "family_name", "given_name", "middle_name", "email", "email_verified", "phone_number", "phone_number_verified", "updated_at", "acr", "amr", "auth_time"},
		GrantTypesSupported:                    []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:      []string{"none", "client_secret_basic"},
		RevocationEndpointAuthMethodsSupported: []string{"none", "client_secret_basic"},
		CodeChallengeMethodsSupported:          []string{"S256"},
	}
}
