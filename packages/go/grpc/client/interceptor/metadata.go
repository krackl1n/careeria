package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}

	for _, value := range md.Get(normalizeMetadataKey(key)) {
		if value != "" {
			return value
		}
	}

	return ""
}

func withMetadataValue(
	ctx context.Context,
	key string,
	value string,
	overwrite bool,
) context.Context {
	if key == "" || value == "" {
		return ctx
	}

	return withMetadata(
		ctx,
		metadata.Pairs(normalizeMetadataKey(key), value),
		overwrite,
	)
}

func withMetadata(ctx context.Context, values metadata.MD, overwrite bool) context.Context {
	if len(values) == 0 {
		return ctx
	}

	current, _ := metadata.FromOutgoingContext(ctx)

	merged := metadata.MD{}
	if current != nil {
		merged = current.Copy()
	}

	changed := false

	for key, metadataValues := range values {
		key = normalizeMetadataKey(key)
		if key == "" || len(metadataValues) == 0 {
			continue
		}

		if !overwrite && len(merged.Get(key)) > 0 {
			continue
		}

		copied := append([]string(nil), metadataValues...)
		merged.Set(key, copied...)
		changed = true
	}

	if !changed {
		return ctx
	}

	return metadata.NewOutgoingContext(ctx, merged)
}

func normalizeMetadataKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
