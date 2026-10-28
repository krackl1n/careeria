package authn

import (
	"context"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryPolicy(t *testing.T) {
	const (
		publicMethod   = "/careeria.identity.v1.OAuthService/ExchangeToken"
		requiredMethod = "/careeria.identity.v1.OAuthService/GetUserInfo"
	)

	tests := []struct {
		name    string
		ctx     context.Context
		method  string
		wantErr codes.Code
	}{
		{name: "public without principal", ctx: context.Background(), method: publicMethod},
		{name: "required with principal", ctx: ctxmeta.WithPrincipal(context.Background(), ctxmeta.Principal{SubjectID: "user-1"}), method: requiredMethod},
		{name: "required without principal", ctx: context.Background(), method: requiredMethod, wantErr: codes.Unauthenticated},
		{name: "unknown method fails closed", ctx: context.Background(), method: "/unknown.Service/Method", wantErr: codes.Unauthenticated},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			err := UnaryPolicy()(test.ctx, test.method, nil, nil, nil, func(
				context.Context,
				string,
				any,
				any,
				*grpc.ClientConn,
				...grpc.CallOption,
			) error {
				called = true
				return nil
			})

			require.Equal(t, test.wantErr, status.Code(err))
			require.Equal(t, test.wantErr == codes.OK, called)
		})
	}
}

func TestStreamPolicyFailsClosed(t *testing.T) {
	called := false
	stream, err := StreamPolicy()(context.Background(), &grpc.StreamDesc{}, nil, "/unknown.Service/Stream", func(
		context.Context,
		*grpc.StreamDesc,
		*grpc.ClientConn,
		string,
		...grpc.CallOption,
	) (grpc.ClientStream, error) {
		called = true
		return nil, nil
	})

	require.Nil(t, stream)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, called)
}
