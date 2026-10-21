package ctxmeta

import "context"

func WithRequestID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, requestIDKey, value)
}

func RequestID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(requestIDKey).(string)
	return value, ok && value != ""
}

func WithCorrelationID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, correlationIDKey, value)
}

func CorrelationID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(correlationIDKey).(string)
	return value, ok && value != ""
}
