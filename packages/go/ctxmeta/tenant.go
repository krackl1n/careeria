package ctxmeta

import "context"

func WithTenantID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, tenantIDKey, value)
}

func TenantID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(tenantIDKey).(string)
	return value, ok && value != ""
}
