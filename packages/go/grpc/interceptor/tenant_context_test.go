package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/grpc/matcher"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryTenantContext(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs(
		DefaultTenantIDMetadataKey,
		"tenant-1",
	))

	response, err := UnaryTenantContext(TenantContextConfig{
		Validate: func(value string) error {
			require.Equal(t, "tenant-1", value)
			return nil
		},
	})(ctx, "request", &grpc.UnaryServerInfo{FullMethod: testFullMethod}, func(ctx context.Context, req any) (any, error) {
		tenantID, ok := ctxmeta.TenantID(ctx)
		require.True(t, ok)
		require.Equal(t, "tenant-1", tenantID)
		return "response", nil
	})

	require.NoError(t, err)
	require.Equal(t, "response", response)
}

func TestTenantContextRejectsMissingOrInvalidMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		cfg  TenantContextConfig
	}{
		{name: "missing", ctx: t.Context()},
		{
			name: "invalid",
			ctx: metadata.NewIncomingContext(t.Context(), metadata.Pairs(
				DefaultTenantIDMetadataKey,
				"invalid",
			)),
			cfg: TenantContextConfig{Validate: func(string) error { return errors.New("invalid") }},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnaryTenantContext(tt.cfg)(tt.ctx, nil, &grpc.UnaryServerInfo{FullMethod: testFullMethod}, func(context.Context, any) (any, error) {
				t.Fatal("handler must not be called")
				return nil, nil
			})
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestTenantContextSkipsUnmatchedMethod(t *testing.T) {
	t.Parallel()

	_, err := UnaryTenantContext(TenantContextConfig{
		Matcher: matcher.MatchMethods("/other.Service/Get"),
	})(t.Context(), nil, &grpc.UnaryServerInfo{FullMethod: testFullMethod}, func(ctx context.Context, req any) (any, error) {
		_, ok := ctxmeta.TenantID(ctx)
		require.False(t, ok)
		return nil, nil
	})

	require.NoError(t, err)
}

func TestStreamTenantContext(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs(
		DefaultTenantIDMetadataKey,
		"tenant-1",
	))
	stream := &fakeServerStream{ctx: ctx}

	err := StreamTenantContext(TenantContextConfig{})(nil, stream, &grpc.StreamServerInfo{FullMethod: testFullMethod}, func(_ any, stream grpc.ServerStream) error {
		tenantID, ok := ctxmeta.TenantID(stream.Context())
		require.True(t, ok)
		require.Equal(t, "tenant-1", tenantID)
		return nil
	})

	require.NoError(t, err)
}
