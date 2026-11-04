package authn

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthConfigValidate(t *testing.T) {
	validJWKSCache := JWKSCacheConfig{TTL: 5 * time.Minute, RefreshTimeout: 2 * time.Second, RefreshCooldown: 10 * time.Second}
	tests := []struct {
		name   string
		config AuthConfig
		want   string
	}{
		{name: "valid", config: AuthConfig{Issuer: "https://identity.example", Audience: []string{"careeria-api"}, JWKSCache: validJWKSCache}},
		{name: "missing issuer", config: AuthConfig{Audience: []string{"careeria-api"}}, want: "JWT issuer is required"},
		{name: "missing audience", config: AuthConfig{Issuer: "https://identity.example"}, want: "JWT audience is required"},
		{name: "empty audience", config: AuthConfig{Issuer: "https://identity.example", Audience: []string{""}}, want: "JWT audience must not be empty"},
		{name: "invalid JWKS cache", config: AuthConfig{Issuer: "https://identity.example", Audience: []string{"careeria-api"}}, want: "validate JWKS cache config: JWKS cache TTL must be positive"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.want)
		})
	}
}
