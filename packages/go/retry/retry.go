package retry

import (
	"context"
	"errors"
	"time"
)

type Operation[T any] func(context.Context) (T, error)

type Config[T any] struct {
	MaxAttempts int
	Backoff     Backoff
	RetryIf     RetryIf[T]
	OnRetry     func(Event[T])
}

type Event[T any] struct {
	Attempt     int
	NextAttempt int
	Result      T
	Err         error
	Delay       time.Duration
}

func Do[T any](
	ctx context.Context,
	cfg Config[T],
	operation Operation[T],
) (T, error) {
	var zero T

	if ctx == nil {
		return zero, ErrNilContext
	}

	if operation == nil {
		return zero, ErrNilOperation
	}

	if err := ctx.Err(); err != nil {
		return zero, err
	}

	cfg = normalizeConfig(cfg)

	var result T
	var err error

	for attempt := 1; ; attempt++ {
		result, err = operation(ctx)

		if !shouldRetry(cfg, attempt, result, err) {
			return result, err
		}

		delay := positiveDuration(cfg.Backoff(attempt))

		if cfg.OnRetry != nil {
			cfg.OnRetry(Event[T]{
				Attempt:     attempt,
				NextAttempt: attempt + 1,
				Result:      result,
				Err:         err,
				Delay:       delay,
			})
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, joinContextError(err, ctxErr)
		}

		if delay <= 0 {
			continue
		}

		if waitErr := wait(ctx, delay); waitErr != nil {
			return result, joinContextError(err, waitErr)
		}
	}
}

func DoError(
	ctx context.Context,
	cfg Config[struct{}],
	operation func(context.Context) error,
) error {
	if operation == nil {
		return ErrNilOperation
	}

	_, err := Do(ctx, cfg, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, operation(ctx)
	})

	return err
}

func normalizeConfig[T any](cfg Config[T]) Config[T] {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	if cfg.Backoff == nil {
		cfg.Backoff = NoBackoff
	}

	if cfg.RetryIf == nil {
		cfg.RetryIf = OnError[T]()
	}

	return cfg
}

func shouldRetry[T any](
	cfg Config[T],
	attempt int,
	result T,
	err error,
) bool {
	if attempt >= cfg.MaxAttempts {
		return false
	}

	return cfg.RetryIf(result, err)
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func joinContextError(err error, ctxErr error) error {
	if ctxErr == nil {
		return err
	}

	if err == nil || errors.Is(err, ctxErr) {
		return ctxErr
	}

	return errors.Join(err, ctxErr)
}
