package authenticationflow

import (
	"testing"

	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestNewFactsValidatesSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		methods []authenticationdomain.Method
		mode    oauthclientdomain.ConsentMode
		covers  bool
		wantErr bool
	}{
		{name: "empty MFA configuration", mode: oauthclientdomain.ConsentModeRemember},
		{name: "configured MFA", methods: []authenticationdomain.Method{authenticationdomain.MethodTOTP, authenticationdomain.MethodWebAuthn}, mode: oauthclientdomain.ConsentModeRequired},
		{name: "covered remembered grant", mode: oauthclientdomain.ConsentModeRemember, covers: true},
		{name: "password is not MFA", methods: []authenticationdomain.Method{authenticationdomain.MethodPassword}, mode: oauthclientdomain.ConsentModeRemember, wantErr: true},
		{name: "duplicate MFA method", methods: []authenticationdomain.Method{authenticationdomain.MethodTOTP, authenticationdomain.MethodTOTP}, mode: oauthclientdomain.ConsentModeRemember, wantErr: true},
		{name: "unknown consent mode", mode: oauthclientdomain.ConsentMode("unknown"), wantErr: true},
		{name: "remembered grant requires remember mode", mode: oauthclientdomain.ConsentModeTrusted, covers: true, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			facts, err := NewFacts(test.methods, test.mode, test.covers)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.methods, facts.AvailableMFAMethods())
			require.Equal(t, len(test.methods) > 0, facts.HasUsableMFA())
		})
	}
}

func TestNewFactsClonesMFAMethods(t *testing.T) {
	t.Parallel()

	methods := []authenticationdomain.Method{authenticationdomain.MethodTOTP}
	facts, err := NewFacts(methods, oauthclientdomain.ConsentModeRemember, false)
	require.NoError(t, err)

	methods[0] = authenticationdomain.MethodWebAuthn
	require.Equal(t, []authenticationdomain.Method{authenticationdomain.MethodTOTP}, facts.AvailableMFAMethods())
}
