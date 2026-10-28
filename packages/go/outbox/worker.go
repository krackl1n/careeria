package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Worker[M any] struct {
	cfg Config

	repository Repository[M]
	publisher  Publisher[M]
	logger     Logger

	drainOnce sync.Once
	drainCh   chan struct{}
	doneCh    chan struct{}
}

func (w *Worker[M]) Name() string { return "outbox" }

func New[M any](
	cfg Config,
	repository Repository[M],
	publisher Publisher[M],
	logger Logger,
) (*Worker[M], error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate outbox worker config: %w", err)
	}

	switch {
	case repository == nil:
		return nil, errors.New("outbox worker repository is required")
	case publisher == nil:
		return nil, errors.New("outbox worker publisher is required")
	case logger == nil:
		return nil, errors.New("outbox worker logger is required")
	}

	return &Worker[M]{
		cfg:        cfg,
		repository: repository,
		publisher:  publisher,
		logger:     logger,
		drainCh:    make(chan struct{}),
		doneCh:     make(chan struct{}),
	}, nil
}

func (w *Worker[M]) Start(ctx context.Context) error {
	defer close(w.doneCh)

	for {
		if ctx.Err() != nil || w.isDraining() {
			return nil
		}

		full := w.process(ctx)

		if ctx.Err() != nil || w.isDraining() {
			return nil
		}

		if full {
			continue
		}

		if !w.wait(ctx) {
			return nil
		}
	}
}

func (w *Worker[M]) Shutdown(ctx context.Context) error {
	w.drainOnce.Do(func() {
		close(w.drainCh)
	})

	select {
	case <-w.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker[M]) process(ctx context.Context) bool {
	entries, err := w.repository.ClaimPending(ctx, w.cfg.BatchSize)
	if err != nil {
		if ctx.Err() == nil {
			w.logger.ErrorContext(
				ctx,
				"claim pending outbox entries failed",
				"error", err,
			)
		}

		return false
	}

	if len(entries) == 0 {
		return false
	}

	published := make([]Entry[M], 0, len(entries))
	finalized := true

	for _, entry := range entries {
		if ctx.Err() != nil {
			return false
		}

		if err := w.publisher.Publish(ctx, entry.Message); err != nil {
			if ctx.Err() != nil {
				return false
			}

			w.logger.ErrorContext(
				ctx,
				"publish outbox entry failed",
				"error", err,
			)

			if markErr := w.repository.MarkFailed(ctx, entry, err); markErr != nil {
				if ctx.Err() != nil {
					return false
				}

				w.logger.ErrorContext(
					ctx,
					"mark outbox entry failed",
					"publish_error", err,
					"error", markErr,
				)

				finalized = false
			}

			continue
		}

		published = append(published, entry)
	}

	if len(published) > 0 {
		if err := w.repository.MarkPublished(ctx, published); err != nil {
			if ctx.Err() == nil {
				w.logger.ErrorContext(
					ctx,
					"mark outbox entries published failed",
					"error", err,
				)
			}

			return false
		}
	}

	return finalized && len(entries) == w.cfg.BatchSize
}

func (w *Worker[M]) wait(ctx context.Context) bool {
	timer := time.NewTimer(w.cfg.PollInterval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-w.drainCh:
		return false
	case <-timer.C:
		return true
	}
}

func (w *Worker[M]) isDraining() bool {
	select {
	case <-w.drainCh:
		return true
	default:
		return false
	}
}
