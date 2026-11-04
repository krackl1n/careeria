package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestUnaryRetry(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil retry function", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc client retry interceptor: nil retry function", func() {
			UnaryRetry(nil)
		})
	})

	t.Run("delegates invocation through retry function", func(t *testing.T) {
		t.Parallel()

		var seenMethod string
		var attempts int

		err := UnaryRetry(func(
			ctx context.Context,
			method string,
			attempt func(context.Context) error,
		) error {
			seenMethod = method
			attempts++

			return attempt(ctx)
		})(
			context.Background(),
			"/toolbox.Service/Create",
			"request",
			"reply",
			nil,
			func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				attempts++
				return nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "/toolbox.Service/Create", seenMethod)
		assert.Equal(t, 2, attempts)
	})
}

func TestStreamRetry(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil retry function", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc client retry interceptor: nil retry function", func() {
			StreamRetry(nil)
		})
	})

	t.Run("delegates stream creation through retry function", func(t *testing.T) {
		t.Parallel()

		desc := &grpc.StreamDesc{ServerStreams: true}
		stream := &mockClientStream{}

		var seenMethod string
		var seenDesc *grpc.StreamDesc

		actualStream, err := StreamRetry(func(
			ctx context.Context,
			method string,
			streamDesc *grpc.StreamDesc,
			open func(context.Context) (grpc.ClientStream, error),
		) (grpc.ClientStream, error) {
			seenMethod = method
			seenDesc = streamDesc

			return open(ctx)
		})(
			context.Background(),
			desc,
			nil,
			"/toolbox.Service/Watch",
			func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				return stream, nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "/toolbox.Service/Watch", seenMethod)
		assert.Same(t, desc, seenDesc)
		assert.Same(t, stream, actualStream)
	})
}

type mockClientStream struct {
	grpc.ClientStream
}
