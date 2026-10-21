package producer

import (
	"context"
	"time"

	"github.com/krackl1n/careeria/kafka"
)

type TimeoutProvider interface {
	Timeout(message kafka.Message) time.Duration
}

type TimeoutProviderFunc func(message kafka.Message) time.Duration

func (f TimeoutProviderFunc) Timeout(message kafka.Message) time.Duration {
	return f(message)
}

type TimeoutConfig struct {
	Provider TimeoutProvider
}

func Timeout(cfg TimeoutConfig) kafka.ProducerMiddleware {
	if cfg.Provider == nil {
		panic("kafka producer timeout middleware: nil provider")
	}

	return func(next kafka.Publisher) kafka.Publisher {
		if next == nil {
			panic("kafka producer timeout middleware: nil publisher")
		}

		return kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			timeout := cfg.Provider.Timeout(message)
			if timeout <= 0 {
				return next.Publish(ctx, message)
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			return next.Publish(ctx, message)
		})
	}
}

func StaticTimeout(timeout time.Duration) TimeoutProvider {
	return TimeoutProviderFunc(func(kafka.Message) time.Duration {
		return timeout
	})
}
