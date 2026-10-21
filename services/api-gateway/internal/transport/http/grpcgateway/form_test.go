package grpcgateway

import (
	"strings"
	"testing"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	"github.com/stretchr/testify/require"
)

func TestFormMarshalerDecodesOAuthRequest(t *testing.T) {
	request := &identityv1.ExchangeTokenRequest{}
	decoder := (FormMarshaler{}).NewDecoder(strings.NewReader(
		"grant_type=authorization_code&client_id=web&code=secret-code&redirect_uri=https%3A%2F%2Fclient.example%2Fcallback&code_verifier=verifier",
	))

	require.NoError(t, decoder.Decode(request))
	require.Equal(t, "authorization_code", request.GetGrantType())
	require.Equal(t, "web", request.GetClientId())
	require.Equal(t, "secret-code", request.GetCode())
	require.Equal(t, "https://client.example/callback", request.GetRedirectUri())
	require.Equal(t, "verifier", request.GetCodeVerifier())
}

func TestFormMarshalerRejectsUnknownField(t *testing.T) {
	request := &identityv1.RevokeTokenRequest{}
	decoder := (FormMarshaler{}).NewDecoder(strings.NewReader("token=value&unexpected=value"))

	require.ErrorContains(t, decoder.Decode(request), "unknown field")
}
