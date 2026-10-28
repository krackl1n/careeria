package consumer

import (
	"context"
	"time"

	"github.com/krackl1n/careeria/kafka"
)

type TimeoutProvider interface {
	Timeout(message kafka.ConsumedMessage) time.Duration
}

type TimeoutProviderFunc func(message kafka.ConsumedMessage) time.Duration

func (f TimeoutProviderFunc) Timeout(message kafka.ConsumedMessage) time.Duration {
	return f(message)
}

type TimeoutConfig struct {
	Provider TimeoutProvider
}

func Timeout(cfg TimeoutConfig) kafka.ConsumerMiddleware {
	if cfg.Provider == nil {
		panic("kafka consumer timeout middleware: nil provider")
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer timeout middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			timeout := cfg.Provider.Timeout(message)
			if timeout <= 0 {
				return next.Handle(ctx, message)
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			err := next.Handle(ctx, message)
			if err != nil {
				return err
			}

			return context.Cause(ctx)
		})
	}
}

func StaticTimeout(timeout time.Duration) TimeoutProvider {
	return TimeoutProviderFunc(func(kafka.ConsumedMessage) time.Duration {
		return timeout
	})
}
