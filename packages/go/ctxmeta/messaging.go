package ctxmeta

import "context"

func WithMessageID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, messageIDKey, value)
}

func MessageID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(messageIDKey).(string)
	return value, ok && value != ""
}

func WithCausationID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, causationIDKey, value)
}

func CausationID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(causationIDKey).(string)
	return value, ok && value != ""
}
