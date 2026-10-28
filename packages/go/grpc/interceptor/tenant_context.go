package interceptor

import (
	"context"
	"errors"
	"strings"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const DefaultTenantIDMetadataKey = "x-tenant-id"

type TenantIDValidator func(string) error

type TenantContextConfig struct {
	MetadataKey string
	Matcher     matcher.Matcher
	Validate    TenantIDValidator
}

func UnaryTenantContext(cfg TenantContextConfig) grpc.UnaryServerInterceptor {
	cfg = normalizeTenantContextConfig(cfg)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !matcher.Matches(cfg.Matcher, info.FullMethod) {
			return handler(ctx, req)
		}

		tenantID, err := tenantIDFromIncomingContext(ctx, cfg)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "tenant context is invalid")
		}

		return handler(ctxmeta.WithTenantID(ctx, tenantID), req)
	}
}

func StreamTenantContext(cfg TenantContextConfig) grpc.StreamServerInterceptor {
	cfg = normalizeTenantContextConfig(cfg)

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !matcher.Matches(cfg.Matcher, info.FullMethod) {
			return handler(srv, stream)
		}

		tenantID, err := tenantIDFromIncomingContext(stream.Context(), cfg)
		if err != nil {
			return status.Error(codes.InvalidArgument, "tenant context is invalid")
		}

		ctx := ctxmeta.WithTenantID(stream.Context(), tenantID)
		return handler(srv, wrapServerStream(stream, ctx))
	}
}

func normalizeTenantContextConfig(cfg TenantContextConfig) TenantContextConfig {
	cfg.MetadataKey = strings.ToLower(strings.TrimSpace(cfg.MetadataKey))
	if cfg.MetadataKey == "" {
		cfg.MetadataKey = DefaultTenantIDMetadataKey
	}
	return cfg
}

func tenantIDFromIncomingContext(ctx context.Context, cfg TenantContextConfig) (string, error) {
	values, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("tenant metadata is required")
	}

	tenantID := ""
	for _, value := range values.Get(cfg.MetadataKey) {
		if value = strings.TrimSpace(value); value != "" {
			tenantID = value
			break
		}
	}
	if tenantID == "" {
		return "", errors.New("tenant metadata is required")
	}
	if cfg.Validate != nil {
		if err := cfg.Validate(tenantID); err != nil {
			return "", err
		}
	}
	return tenantID, nil
}
