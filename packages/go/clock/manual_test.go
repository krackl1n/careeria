package clock_test

import (
	"testing"
	"time"

	"github.com/krackl1n/careeria/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualNow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 17, 12, 30, 0, 0, time.UTC)
	c := clock.NewManual(now)

	require.Equal(t, now, c.Now())
}

func TestManualSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial time.Time
		next    time.Time
	}{
		{
			name:    "set future time",
			initial: time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
			next:    time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC),
		},
		{
			name:    "set past time",
			initial: time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
			next:    time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC),
		},
		{
			name:    "set zero time",
			initial: time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
			next:    time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := clock.NewManual(tt.initial)

			c.Set(tt.next)

			assert.Equal(t, tt.next, c.Now())
		})
	}
}

func TestManualAdvance(t *testing.T) {
	t.Parallel()

	initial := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		duration time.Duration
		expected time.Time
	}{
		{
			name:     "advance forward",
			duration: 5 * time.Minute,
			expected: initial.Add(5 * time.Minute),
		},
		{
			name:     "advance backward",
			duration: -5 * time.Minute,
			expected: initial.Add(-5 * time.Minute),
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: initial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := clock.NewManual(initial)

			c.Advance(tt.duration)

			assert.Equal(t, tt.expected, c.Now())
		})
	}
}

func TestManualImplementsClock(t *testing.T) {
	t.Parallel()

	var c clock.Clock = clock.NewManual(time.Time{})

	assert.NotNil(t, c)
}

func TestManualConcurrentAccess(t *testing.T) {
	t.Parallel()

	initial := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	c := clock.NewManual(initial)

	const goroutines = 100

	done := make(chan struct{}, goroutines)

	for range goroutines {
		go func() {
			c.Advance(time.Second)
			_ = c.Now()
			done <- struct{}{}
		}()
	}

	for range goroutines {
		<-done
	}

	require.Equal(t, initial.Add(goroutines*time.Second), c.Now())
}
