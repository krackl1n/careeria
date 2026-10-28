package interceptor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	DefaultRequestIDMetadataKey     = "x-request-id"
	DefaultCorrelationIDMetadataKey = "x-correlation-id"
)

type RequestIDGenerator interface {
	Generate() string
}

type RequestIDConfig struct {
	MetadataKey            string
	CorrelationMetadataKey string
	Generator              RequestIDGenerator
}

func UnaryRequestID(cfg RequestIDConfig) grpc.UnaryServerInterceptor {
	cfg = normalizeRequestIDConfig(cfg)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		requestID := requestIDFromMetadata(ctx, cfg.MetadataKey)
		if requestID == "" {
			requestID = generateRequestID(cfg.Generator)
		}

		if requestID == "" {
			return handler(ctx, req)
		}

		ctx = withRequestMetadata(ctx, requestID, cfg.CorrelationMetadataKey)

		_ = grpc.SetHeader(ctx, metadata.Pairs(cfg.MetadataKey, requestID))

		return handler(ctx, req)
	}
}

func StreamRequestID(cfg RequestIDConfig) grpc.StreamServerInterceptor {
	cfg = normalizeRequestIDConfig(cfg)

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()

		requestID := requestIDFromMetadata(ctx, cfg.MetadataKey)
		if requestID == "" {
			requestID = generateRequestID(cfg.Generator)
		}

		if requestID == "" {
			return handler(srv, stream)
		}

		ctx = withRequestMetadata(ctx, requestID, cfg.CorrelationMetadataKey)

		_ = stream.SetHeader(metadata.Pairs(cfg.MetadataKey, requestID))

		return handler(srv, wrapServerStream(stream, ctx))
	}
}

func normalizeRequestIDConfig(cfg RequestIDConfig) RequestIDConfig {
	if cfg.MetadataKey == "" {
		cfg.MetadataKey = DefaultRequestIDMetadataKey
	}

	cfg.MetadataKey = strings.ToLower(cfg.MetadataKey)
	if cfg.CorrelationMetadataKey == "" {
		cfg.CorrelationMetadataKey = DefaultCorrelationIDMetadataKey
	}
	cfg.CorrelationMetadataKey = strings.ToLower(cfg.CorrelationMetadataKey)

	return cfg
}

func requestIDFromMetadata(ctx context.Context, key string) string {
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

func requestIDForLog(ctx context.Context) string {
	requestID, ok := ctxmeta.RequestID(ctx)
	if ok {
		return requestID
	}

	return requestIDFromMetadata(ctx, DefaultRequestIDMetadataKey)
}

func withRequestMetadata(ctx context.Context, requestID, correlationKey string) context.Context {
	ctx = ctxmeta.WithRequestID(ctx, requestID)
	correlationID := requestIDFromMetadata(ctx, correlationKey)
	if correlationID == "" {
		correlationID = requestID
	}

	return ctxmeta.WithCorrelationID(ctx, correlationID)
}

func generateRequestID(generator RequestIDGenerator) string {
	if generator != nil {
		return generator.Generate()
	}

	var value [16]byte

	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}

	return hex.EncodeToString(value[:])
}
