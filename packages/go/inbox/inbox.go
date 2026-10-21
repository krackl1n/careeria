package inbox

import (
	"context"
	"fmt"
)

type Inbox[M any] struct {
	txManager  TxManager
	repository Repository[M]
}

func New[M any](
	txManager TxManager,
	repository Repository[M],
) (*Inbox[M], error) {
	if txManager == nil {
		return nil, ErrNilTxManager
	}

	if repository == nil {
		return nil, ErrNilRepository
	}

	return &Inbox[M]{
		txManager:  txManager,
		repository: repository,
	}, nil
}

func (i *Inbox[M]) WithInbox(
	ctx context.Context,
	message M,
	handler Handler,
) error {
	if handler == nil {
		return ErrNilHandler
	}

	return i.txManager.Do(ctx, func(ctx context.Context) error {
		inserted, err := i.repository.Insert(ctx, message)
		if err != nil {
			return fmt.Errorf("insert inbox message: %w", err)
		}

		if !inserted {
			return nil
		}

		if err := handler(ctx); err != nil {
			return fmt.Errorf("handle inbox message: %w", err)
		}

		return nil
	})
}
