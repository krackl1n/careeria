package outbox

import "context"

type Entry[M any] struct {
	ID         string
	ClaimToken string
	Message    M
}

type Repository[M any] interface {
	ClaimPending(ctx context.Context, limit int) ([]Entry[M], error)
	MarkPublished(ctx context.Context, entries []Entry[M]) error
	MarkFailed(ctx context.Context, entry Entry[M], cause error) error
}

type Publisher[M any] interface {
	Publish(ctx context.Context, message M) error
}

type Logger interface {
	ErrorContext(ctx context.Context, msg string, args ...any)
}
