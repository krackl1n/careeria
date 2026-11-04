package interceptor

import (
	"context"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"

	"google.golang.org/grpc"
)

type TimeoutProvider func(fullMethod string) time.Duration

type MethodTimeoutDeps struct {
	Provider TimeoutProvider
	Matcher  matcher.Matcher
}

func UnaryMethodTimeout(cfg MethodTimeoutDeps) grpc.UnaryClientInterceptor {
	if cfg.Provider == nil {
		panic("grpc client method timeout interceptor: nil timeout provider")
	}

	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if !matcher.Matches(cfg.Matcher, method) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		timeoutCtx, cancel := withMethodTimeout(ctx, cfg.Provider(method))
		if cancel == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		defer cancel()

		return invoker(
			timeoutCtx,
			method,
			req,
			reply,
			cc,
			opts...,
		)
	}
}

func StreamMethodTimeout(cfg MethodTimeoutDeps) grpc.StreamClientInterceptor {
	if cfg.Provider == nil {
		panic("grpc client method timeout interceptor: nil timeout provider")
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if !matcher.Matches(cfg.Matcher, method) {
			return streamer(ctx, desc, cc, method, opts...)
		}

		timeoutCtx, cancel := withMethodTimeout(ctx, cfg.Provider(method))
		if cancel == nil {
			return streamer(ctx, desc, cc, method, opts...)
		}

		stream, err := streamer(
			timeoutCtx,
			desc,
			cc,
			method,
			opts...,
		)
		if err != nil {
			cancel()
			return nil, err
		}

		return wrapClientStream(stream, ctx, desc, func(error) {
			cancel()
		}), nil
	}
}

func withMethodTimeout(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, nil
	}

	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= timeout {
		return ctx, nil
	}

	return context.WithTimeout(ctx, timeout)
}
