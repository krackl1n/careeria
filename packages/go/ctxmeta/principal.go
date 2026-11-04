package ctxmeta

import "context"

type Principal struct {
	SubjectID string
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey).(Principal)
	if !ok {
		return Principal{}, false
	}

	return principal, true
}
