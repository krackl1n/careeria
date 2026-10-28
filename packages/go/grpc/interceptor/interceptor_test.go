package interceptor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/grpc/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const testFullMethod = "/toolbox.TestService/Create"

type authenticatorFunc func(context.Context, string) (ctxmeta.Principal, error)

func (fn authenticatorFunc) Authenticate(ctx context.Context, fullMethod string) (ctxmeta.Principal, error) {
	return fn(ctx, fullMethod)
}

type authorizerFunc func(context.Context, string) error

func (fn authorizerFunc) Authorize(ctx context.Context, fullMethod string) error {
	return fn(ctx, fullMethod)
}

type timeoutProviderFunc func(string) time.Duration

func (fn timeoutProviderFunc) Timeout(fullMethod string) time.Duration {
	return fn(fullMethod)
}

type idempotencyManagerFunc func(
	context.Context,
	string,
	string,
	any,
	IdempotencyOperation,
) (any, error)

func (fn idempotencyManagerFunc) Execute(
	ctx context.Context,
	scope string,
	key string,
	request any,
	operation IdempotencyOperation,
) (any, error) {
	return fn(ctx, scope, key, request, operation)
}

type errorResolverFunc func(error) (ResolvedError, bool)

func (fn errorResolverFunc) Resolve(err error) (ResolvedError, bool) {
	return fn(err)
}

type panicHandlerFunc func(context.Context, string, any) error

func (fn panicHandlerFunc) Handle(ctx context.Context, fullMethod string, recovered any) error {
	return fn(ctx, fullMethod, recovered)
}

type requestIDGeneratorFunc func() string

func (fn requestIDGeneratorFunc) Generate() string {
	return fn()
}

type fakeServerStream struct {
	grpc.ServerStream

	ctx     context.Context
	headers metadata.MD
	trailer metadata.MD

	recv func(any) error
}

func (s *fakeServerStream) SetHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (s *fakeServerStream) SendHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (s *fakeServerStream) SetTrailer(md metadata.MD) {
	s.trailer = metadata.Join(s.trailer, md)
}

func (s *fakeServerStream) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}

	return s.ctx
}

func (s *fakeServerStream) SendMsg(any) error {
	return nil
}

func (s *fakeServerStream) RecvMsg(message any) error {
	if s.recv == nil {
		return io.EOF
	}

	return s.recv(message)
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	t.Run("unary uses incoming request id", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				DefaultRequestIDMetadataKey, "req-123",
				DefaultCorrelationIDMetadataKey, "corr-123",
			),
		)

		resp, err := UnaryRequestID(RequestIDConfig{})(
			ctx,
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				requestID, _ := ctxmeta.RequestID(ctx)
				assert.Equal(t, "req-123", requestID)
				correlationID, _ := ctxmeta.CorrelationID(ctx)
				assert.Equal(t, "corr-123", correlationID)
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("unary generates missing request id", func(t *testing.T) {
		t.Parallel()

		resp, err := UnaryRequestID(RequestIDConfig{
			Generator: requestIDGeneratorFunc(func() string {
				return "generated-123"
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				requestID, _ := ctxmeta.RequestID(ctx)
				assert.Equal(t, "generated-123", requestID)
				correlationID, _ := ctxmeta.CorrelationID(ctx)
				assert.Equal(t, "generated-123", correlationID)
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("stream stores request id and sends header", func(t *testing.T) {
		t.Parallel()

		stream := &fakeServerStream{
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(DefaultRequestIDMetadataKey, "req-stream"),
			),
		}

		err := StreamRequestID(RequestIDConfig{})(
			nil,
			stream,
			&grpc.StreamServerInfo{FullMethod: testFullMethod},
			func(srv any, stream grpc.ServerStream) error {
				requestID, _ := ctxmeta.RequestID(stream.Context())
				assert.Equal(t, "req-stream", requestID)
				return nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"req-stream"}, stream.headers.Get(DefaultRequestIDMetadataKey))
	})
}

func TestAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil authenticator", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc authentication interceptor: nil authenticator", func() {
			UnaryAuthentication(AuthenticationConfig{})
		})
		require.PanicsWithValue(t, "grpc authentication interceptor: nil authenticator", func() {
			StreamAuthentication(AuthenticationConfig{})
		})
	})

	t.Run("unary authenticates and stores principal", func(t *testing.T) {
		t.Parallel()

		resp, err := UnaryAuthentication(AuthenticationConfig{
			Authenticator: authenticatorFunc(func(ctx context.Context, fullMethod string) (ctxmeta.Principal, error) {
				assert.Equal(t, testFullMethod, fullMethod)
				return ctxmeta.Principal{SubjectID: "user-1"}, nil
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				principal, ok := ctxmeta.PrincipalFromContext(ctx)
				require.True(t, ok)
				assert.Equal(t, "user-1", principal.SubjectID)
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("skips unmatched method", func(t *testing.T) {
		t.Parallel()

		called := false

		resp, err := UnaryAuthentication(AuthenticationConfig{
			Matcher: matcher.MatchMethods("/other.Service/Get"),
			Authenticator: authenticatorFunc(func(context.Context, string) (ctxmeta.Principal, error) {
				called = true
				return ctxmeta.Principal{SubjectID: "user-1"}, nil
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.False(t, called)
	})

	t.Run("normalizes non-status error", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryAuthentication(AuthenticationConfig{
			Authenticator: authenticatorFunc(func(context.Context, string) (ctxmeta.Principal, error) {
				return ctxmeta.Principal{}, errors.New("token parse failed")
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				t.Fatal("handler must not be called")
				return nil, nil
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Equal(t, "unauthenticated", status.Convert(err).Message())
	})

	t.Run("stream authenticates and stores principal", func(t *testing.T) {
		t.Parallel()

		stream := &fakeServerStream{ctx: context.Background()}

		err := StreamAuthentication(AuthenticationConfig{
			Authenticator: authenticatorFunc(func(context.Context, string) (ctxmeta.Principal, error) {
				return ctxmeta.Principal{SubjectID: "user-1"}, nil
			}),
		})(
			nil,
			stream,
			&grpc.StreamServerInfo{FullMethod: testFullMethod},
			func(srv any, stream grpc.ServerStream) error {
				principal, ok := ctxmeta.PrincipalFromContext(stream.Context())
				require.True(t, ok)
				assert.Equal(t, "user-1", principal.SubjectID)
				return nil
			},
		)

		require.NoError(t, err)
	})
}

func TestAuthorization(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil authorizer", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc authorization interceptor: nil authorizer", func() {
			UnaryAuthorization(AuthorizationConfig{})
		})
		require.PanicsWithValue(t, "grpc authorization interceptor: nil authorizer", func() {
			StreamAuthorization(AuthorizationConfig{})
		})
	})

	t.Run("allows authorized unary request", func(t *testing.T) {
		t.Parallel()

		resp, err := UnaryAuthorization(AuthorizationConfig{
			Authorizer: authorizerFunc(func(ctx context.Context, fullMethod string) error {
				assert.Equal(t, testFullMethod, fullMethod)
				return nil
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("normalizes non-status error", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryAuthorization(AuthorizationConfig{
			Authorizer: authorizerFunc(func(context.Context, string) error {
				return errors.New("not allowed")
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				t.Fatal("handler must not be called")
				return nil, nil
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Equal(t, "permission denied", status.Convert(err).Message())
	})
}

func TestIdempotency(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil manager", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc idempotency interceptor: nil manager", func() {
			UnaryIdempotency(IdempotencyConfig{})
		})
	})

	t.Run("returns invalid argument when required key is missing", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryIdempotency(IdempotencyConfig{
			Required: true,
			Manager: idempotencyManagerFunc(func(
				context.Context,
				string,
				string,
				any,
				IdempotencyOperation,
			) (any, error) {
				t.Fatal("manager must not be called")
				return nil, nil
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				t.Fatal("handler must not be called")
				return nil, nil
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Equal(t, "idempotency key is required", status.Convert(err).Message())
	})

	t.Run("executes manager when key exists", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(DefaultIdempotencyMetadataKey, "idem-123"),
		)

		resp, err := UnaryIdempotency(IdempotencyConfig{
			Manager: idempotencyManagerFunc(func(
				ctx context.Context,
				scope string,
				key string,
				request any,
				operation IdempotencyOperation,
			) (any, error) {
				assert.Equal(t, testFullMethod, scope)
				assert.Equal(t, "idem-123", key)
				assert.Equal(t, "request", request)

				resp, err := operation(ctx)
				require.NoError(t, err)

				return resp, nil
			}),
		})(
			ctx,
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}

type validRequest struct {
	err error
}

func (r validRequest) Validate() error {
	return r.err
}

func TestValidation(t *testing.T) {
	t.Parallel()

	t.Run("unary rejects invalid request", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryValidation()(
			context.Background(),
			validRequest{err: errors.New("invalid")},
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				t.Fatal("handler must not be called")
				return nil, nil
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Equal(t, "request validation failed", status.Convert(err).Message())
	})

	t.Run("stream validates received messages", func(t *testing.T) {
		t.Parallel()

		stream := &fakeServerStream{
			ctx: context.Background(),
			recv: func(message any) error {
				req := message.(*validRequest)
				req.err = errors.New("invalid")
				return nil
			},
		}

		err := StreamValidation()(
			nil,
			stream,
			&grpc.StreamServerInfo{FullMethod: testFullMethod},
			func(srv any, stream grpc.ServerStream) error {
				var req validRequest
				return stream.RecvMsg(&req)
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestMethodTimeout(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil provider", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc method timeout interceptor: nil provider", func() {
			UnaryMethodTimeout(MethodTimeoutConfig{})
		})
		require.PanicsWithValue(t, "grpc method timeout interceptor: nil provider", func() {
			StreamMethodTimeout(MethodTimeoutConfig{})
		})
	})

	t.Run("unary applies method timeout", func(t *testing.T) {
		t.Parallel()

		resp, err := UnaryMethodTimeout(MethodTimeoutConfig{
			Provider: timeoutProviderFunc(func(fullMethod string) time.Duration {
				assert.Equal(t, testFullMethod, fullMethod)
				return time.Minute
			}),
		})(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, time.Second)
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("stream applies method timeout", func(t *testing.T) {
		t.Parallel()

		stream := &fakeServerStream{ctx: context.Background()}

		err := StreamMethodTimeout(MethodTimeoutConfig{
			Provider: timeoutProviderFunc(func(string) time.Duration {
				return time.Minute
			}),
		})(
			nil,
			stream,
			&grpc.StreamServerInfo{FullMethod: testFullMethod},
			func(srv any, stream grpc.ServerStream) error {
				_, ok := stream.Context().Deadline()
				require.True(t, ok)
				return nil
			},
		)

		require.NoError(t, err)
	})
}

func TestErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("custom resolver has priority", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryErrorMapping(nil, errorResolverFunc(func(err error) (ResolvedError, bool) {
			return ResolvedError{
				Code:    codes.NotFound,
				Message: "resource not found",
			}, true
		}))(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("domain failure")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, "resource not found", status.Convert(err).Message())
	})

	t.Run("adds resolver details to status", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryErrorMapping(nil, errorResolverFunc(func(error) (ResolvedError, bool) {
			return ResolvedError{
				Code:    codes.InvalidArgument,
				Message: "OAuth grant is invalid",
				Details: []proto.Message{&errdetails.ErrorInfo{
					Reason: "oauth_invalid_grant",
					Domain: "careeria.identity",
				}},
			}, true
		}))(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(context.Context, any) (any, error) {
				return nil, errors.New("domain failure")
			},
		)

		require.Equal(t, codes.InvalidArgument, status.Code(err))
		details := status.Convert(err).Details()
		require.Len(t, details, 1)
		info, ok := details[0].(*errdetails.ErrorInfo)
		require.True(t, ok)
		require.Equal(t, "oauth_invalid_grant", info.GetReason())
		require.Equal(t, "careeria.identity", info.GetDomain())
	})

	t.Run("sanitizes internal resolver result", func(t *testing.T) {
		t.Parallel()
		var logs bytes.Buffer

		_, err := UnaryErrorMapping(slog.New(slog.NewJSONHandler(&logs, nil)), errorResolverFunc(func(err error) (ResolvedError, bool) {
			return ResolvedError{
				Code:    codes.Internal,
				Message: "database password leaked",
			}, true
		}))(
			ctxmeta.WithRequestID(context.Background(), "req-123"),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("domain failure")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "internal server error", status.Convert(err).Message())
		assert.Contains(t, logs.String(), "mapped grpc server error")
		assert.Contains(t, logs.String(), "domain failure")
		assert.Contains(t, logs.String(), "req-123")
	})

	t.Run("preserves status errors", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryErrorMapping(nil, nil)(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, status.Error(codes.AlreadyExists, "already exists")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		assert.Equal(t, "already exists", status.Convert(err).Message())
	})

	t.Run("preserves status error details", func(t *testing.T) {
		t.Parallel()

		grpcStatus, statusErr := status.New(
			codes.InvalidArgument,
			"request validation failed",
		).WithDetails(&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "title",
				Reason:      "invalid",
				Description: "must not be empty",
			},
		}})
		require.NoError(t, statusErr)

		_, err := UnaryErrorMapping(nil, errorResolverFunc(func(error) (ResolvedError, bool) {
			t.Fatal("error resolver must not receive an already resolved status error")
			return ResolvedError{}, false
		}))(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(context.Context, any) (any, error) {
				return nil, grpcStatus.Err()
			},
		)

		require.Equal(t, codes.InvalidArgument, status.Code(err))
		details := status.Convert(err).Details()
		require.Len(t, details, 1)
		badRequest, ok := details[0].(*errdetails.BadRequest)
		require.True(t, ok)
		violations := badRequest.GetFieldViolations()
		require.Len(t, violations, 1)
		require.Equal(t, "title", violations[0].GetField())
		require.Equal(t, "invalid", violations[0].GetReason())
		require.Equal(t, "must not be empty", violations[0].GetDescription())
	})

	t.Run("maps context errors", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryErrorMapping(nil, nil)(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, context.DeadlineExceeded
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assert.Equal(t, "deadline exceeded", status.Convert(err).Message())
	})

	t.Run("unknown error falls back to internal and logs original error", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer

		_, err := UnaryErrorMapping(
			slog.New(slog.NewJSONHandler(&logs, nil)),
			nil,
		)(
			ctxmeta.WithRequestID(context.Background(), "req-123"),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("database failed")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "internal server error", status.Convert(err).Message())
		assert.Contains(t, logs.String(), "database failed")
		assert.Contains(t, logs.String(), "req-123")
	})
}

func TestRecovery(t *testing.T) {
	t.Parallel()

	t.Run("unary recovers panic as internal error", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer

		_, err := UnaryRecovery(
			slog.New(slog.NewJSONHandler(&logs, nil)),
			nil,
		)(
			ctxmeta.WithRequestID(context.Background(), "req-123"),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				panic("boom")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "internal server error", status.Convert(err).Message())
		assert.Contains(t, logs.String(), "grpc panic recovered")
		assert.Contains(t, logs.String(), "boom")
	})

	t.Run("panic handler can map recovered panic", func(t *testing.T) {
		t.Parallel()

		_, err := UnaryRecovery(nil, panicHandlerFunc(func(
			ctx context.Context,
			fullMethod string,
			recovered any,
		) error {
			assert.Equal(t, testFullMethod, fullMethod)
			assert.Equal(t, "boom", recovered)
			return status.Error(codes.Unavailable, "temporary unavailable")
		}))(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				panic("boom")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
		assert.Equal(t, "temporary unavailable", status.Convert(err).Message())
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	ctx := peer.NewContext(
		ctxmeta.WithRequestID(context.Background(), "req-123"),
		&peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50051}},
	)

	resp, err := UnaryLogging(slog.New(slog.NewJSONHandler(&logs, nil)))(
		ctx,
		"request",
		&grpc.UnaryServerInfo{FullMethod: testFullMethod},
		func(ctx context.Context, req any) (any, error) {
			return "response", nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.Contains(t, logs.String(), "grpc request completed")
	assert.Contains(t, logs.String(), "toolbox.TestService")
	assert.Contains(t, logs.String(), "Create")
	assert.Contains(t, logs.String(), "req-123")
	assert.Contains(t, logs.String(), "127.0.0.1:50051")
}
