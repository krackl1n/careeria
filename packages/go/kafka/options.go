package kafka

type ProducerOption func(*producerOptions)

type producerOptions struct {
	middlewares []ProducerMiddleware
}

func WithProducerMiddleware(
	middlewares ...ProducerMiddleware,
) ProducerOption {
	return func(options *producerOptions) {
		options.middlewares = append(
			options.middlewares,
			middlewares...,
		)
	}
}

type ConsumerOption func(*consumerOptions)

type consumerOptions struct {
	middlewares []ConsumerMiddleware
}

func WithConsumerMiddleware(
	middlewares ...ConsumerMiddleware,
) ConsumerOption {
	return func(options *consumerOptions) {
		options.middlewares = append(
			options.middlewares,
			middlewares...,
		)
	}
}
