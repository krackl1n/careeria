package grpcserver

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	require.EqualError(t, Config{}.Validate(), "grpc server addr is required")
	require.NoError(t, Config{Addr: ":8080"}.Validate())
}

func TestOptions(t *testing.T) {
	t.Parallel()

	var opts options

	unary := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	}
	stream := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	WithUnaryInterceptors(unary)(&opts)
	WithStreamInterceptors(stream)(&opts)
	WithServerOptions(grpc.ConnectionTimeout(time.Second))(&opts)

	require.Len(t, opts.unaryInterceptors, 1)
	require.Len(t, opts.streamInterceptors, 1)
	require.Len(t, opts.serverOptions, 1)
}

func TestNewAndShutdown(t *testing.T) {
	t.Parallel()

	t.Run("invalid config", func(t *testing.T) {
		t.Parallel()
		server, err := New(Config{})
		require.Nil(t, server)
		require.EqualError(t, err, "validate grpc config: grpc server addr is required")
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		server, err := New(Config{
			Addr:              ":0",
			EnableHealth:      true,
			EnableReflection:  true,
			ConnectionTimeout: time.Second,
			MaxRecvMsgSize:    1024,
			MaxSendMsgSize:    2048,
		})
		require.NoError(t, err)
		require.NotNil(t, server)
		require.NotNil(t, server.GRPC())
		require.NoError(t, server.Shutdown(context.Background()))
	})
}
