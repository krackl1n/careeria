package inbox

import "context"

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Repository[M any] interface {
	Insert(ctx context.Context, message M) (inserted bool, err error)
}

type Handler func(ctx context.Context) error
