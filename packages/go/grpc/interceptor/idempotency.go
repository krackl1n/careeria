package interceptor

import (
	"context"
	"strings"

	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const DefaultIdempotencyMetadataKey = "idempotency-key"

type IdempotencyOperation func(context.Context) (any, error)

type IdempotencyManager interface {
	Execute(
		ctx context.Context,
		scope string,
		key string,
		request any,
		operation IdempotencyOperation,
	) (any, error)
}

type IdempotencyConfig struct {
	Manager IdempotencyManager
	Matcher matcher.Matcher

	MetadataKey string
	Required    bool
}

func UnaryIdempotency(cfg IdempotencyConfig) grpc.UnaryServerInterceptor {
	if cfg.Manager == nil {
		panic("grpc idempotency interceptor: nil manager")
	}

	if cfg.MetadataKey == "" {
		cfg.MetadataKey = DefaultIdempotencyMetadataKey
	}

	cfg.MetadataKey = strings.ToLower(cfg.MetadataKey)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !matcher.Matches(cfg.Matcher, info.FullMethod) {
			return handler(ctx, req)
		}

		key := metadataValue(ctx, cfg.MetadataKey)
		if key == "" {
			if cfg.Required {
				return nil, status.Error(
					codes.InvalidArgument,
					"idempotency key is required",
				)
			}

			return handler(ctx, req)
		}

		return cfg.Manager.Execute(
			ctx,
			info.FullMethod,
			key,
			req,
			func(operationCtx context.Context) (any, error) {
				return handler(operationCtx, req)
			},
		)
	}
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
