package identity

import (
	"context"
	"errors"
	"testing"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type metadataClientStub struct {
	keys []*structpb.Struct
	err  error
}

func (metadataClientStub) GetOpenIDConfiguration(
	context.Context,
	*identityv1.GetOpenIDConfigurationRequest,
	...grpc.CallOption,
) (*identityv1.GetOpenIDConfigurationResponse, error) {
	return nil, errors.New("not implemented")
}

func (s metadataClientStub) GetJSONWebKeySet(
	context.Context,
	*identityv1.GetJSONWebKeySetRequest,
	...grpc.CallOption,
) (*identityv1.GetJSONWebKeySetResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &identityv1.GetJSONWebKeySetResponse{Keys: s.keys}, nil
}

func TestLoadJWKS(t *testing.T) {
	key := testJWK(t, "key-1")
	clients := &Clients{Metadata: metadataClientStub{keys: []*structpb.Struct{key}}}

	result, err := clients.LoadJWKS(t.Context())

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.JSONEq(t, `{"kty":"OKP","crv":"Ed25519","kid":"key-1","x":"public"}`, string(result["key-1"]))
}

func TestLoadJWKSRejectsUnavailableOrInvalidKeySet(t *testing.T) {
	tests := []struct {
		name    string
		clients *Clients
		want    string
	}{
		{name: "missing client", clients: &Clients{}, want: "identity metadata client is required"},
		{name: "request failure", clients: &Clients{Metadata: metadataClientStub{err: errors.New("unavailable")}}, want: "get JSON Web Key Set: unavailable"},
		{name: "empty key set", clients: &Clients{Metadata: metadataClientStub{}}, want: "JSON Web Key Set is empty"},
		{name: "nil key", clients: &Clients{Metadata: metadataClientStub{keys: []*structpb.Struct{nil}}}, want: "JSON Web Key Set contains an empty key"},
		{name: "missing key ID", clients: &Clients{Metadata: metadataClientStub{keys: []*structpb.Struct{testJWK(t, "")}}}, want: "JSON Web Key Set contains a key without kid"},
		{name: "duplicate key ID", clients: &Clients{Metadata: metadataClientStub{keys: []*structpb.Struct{testJWK(t, "duplicate"), testJWK(t, "duplicate")}}}, want: `JSON Web Key Set contains duplicate key "duplicate"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.clients.LoadJWKS(t.Context())
			require.EqualError(t, err, test.want)
		})
	}
}

func testJWK(t *testing.T, keyID string) *structpb.Struct {
	t.Helper()
	key, err := structpb.NewStruct(map[string]any{
		"kty": "OKP",
		"crv": "Ed25519",
		"kid": keyID,
		"x":   "public",
	})
	require.NoError(t, err)
	return key
}
