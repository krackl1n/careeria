package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"
)

var ErrHandlerPanic = errors.New("kafka consumer handler panic")

type PanicHandler interface {
	Handle(ctx context.Context, message kafka.ConsumedMessage, recovered any) error
}

func Recovery(logger *slog.Logger, panicHandler PanicHandler) kafka.ConsumerMiddleware {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer recovery middleware: nil handler")
		}

		return kafka.HandlerFunc(func(
			ctx context.Context,
			message kafka.ConsumedMessage,
		) (err error) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				messageID, _ := ctxmeta.MessageID(ctx)
				correlationID, _ := ctxmeta.CorrelationID(ctx)
				logger.ErrorContext(
					ctx,
					"kafka consumer panic recovered",
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
					"message_id", messageID,
					"correlation_id", correlationID,
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)

				if panicHandler != nil {
					err = panicHandler.Handle(ctx, message, recovered)
					if err != nil {
						return
					}
				}

				err = fmt.Errorf("%w: %v", ErrHandlerPanic, recovered)
			}()

			return next.Handle(ctx, message)
		})
	}
}
