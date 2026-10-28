package kafka

import "context"

type ProducerMiddleware func(next Publisher) Publisher

type ConsumerMiddleware func(next Handler) Handler

type PublisherFunc func(ctx context.Context, message Message) error

func (f PublisherFunc) Publish(
	ctx context.Context,
	message Message,
) error {
	return f(ctx, message)
}

func applyProducerMiddleware(
	publisher Publisher,
	middlewares []ProducerMiddleware,
) Publisher {
	result := publisher

	for index := len(middlewares) - 1; index >= 0; index-- {
		middleware := middlewares[index]

		if middleware != nil {
			result = middleware(result)
		}
	}

	return result
}

func applyConsumerMiddleware(
	handler Handler,
	middlewares []ConsumerMiddleware,
) Handler {
	result := handler

	for index := len(middlewares) - 1; index >= 0; index-- {
		middleware := middlewares[index]

		if middleware != nil {
			result = middleware(result)
		}
	}

	return result
}
