package retry

type RetryIf[T any] func(result T, err error) bool

func OnError[T any]() RetryIf[T] {
	return func(_ T, err error) bool {
		return err != nil
	}
}

func OnResult[T any](fn func(T) bool) RetryIf[T] {
	return func(result T, err error) bool {
		return err == nil && fn != nil && fn(result)
	}
}

func Never[T any]() RetryIf[T] {
	return func(T, error) bool {
		return false
	}
}

func Any[T any](rules ...RetryIf[T]) RetryIf[T] {
	return func(result T, err error) bool {
		for _, rule := range rules {
			if rule != nil && rule(result, err) {
				return true
			}
		}

		return false
	}
}

func All[T any](rules ...RetryIf[T]) RetryIf[T] {
	return func(result T, err error) bool {
		matched := false

		for _, rule := range rules {
			if rule == nil {
				continue
			}

			matched = true
			if !rule(result, err) {
				return false
			}
		}

		return matched
	}
}
