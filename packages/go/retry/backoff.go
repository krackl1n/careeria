package retry

import (
	"math"
	"math/bits"
	"math/rand"
	"time"
)

type Backoff func(attempt int) time.Duration

func NoBackoff(int) time.Duration {
	return 0
}

func Constant(delay time.Duration) Backoff {
	return func(int) time.Duration {
		return positiveDuration(delay)
	}
}

func Exponential(base time.Duration, max time.Duration) Backoff {
	base = positiveDuration(base)
	max = positiveDuration(max)

	return func(attempt int) time.Duration {
		if attempt <= 1 || base == 0 {
			return base
		}

		power := attempt - 1
		if power >= bits.UintSize-2 {
			return capDuration(base, max)
		}

		if base > time.Duration(math.MaxInt64>>power) {
			return capDuration(time.Duration(math.MaxInt64), max)
		}

		return capDuration(base*time.Duration(1<<power), max)
	}
}

func WithJitter(backoff Backoff, ratio float64, random func() float64) Backoff {
	if backoff == nil {
		backoff = NoBackoff
	}

	if ratio <= 0 {
		return backoff
	}

	if ratio > 1 {
		ratio = 1
	}

	if random == nil {
		random = rand.Float64
	}

	return func(attempt int) time.Duration {
		delay := positiveDuration(backoff(attempt))
		if delay == 0 {
			return 0
		}

		value := random()
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}

		factor := 1 - ratio + value*(2*ratio)
		if factor < 0 {
			return 0
		}

		return time.Duration(float64(delay) * factor)
	}
}

func positiveDuration(value time.Duration) time.Duration {
	if value <= 0 {
		return 0
	}

	return value
}

func capDuration(value time.Duration, max time.Duration) time.Duration {
	if max > 0 && value > max {
		return max
	}

	return value
}
