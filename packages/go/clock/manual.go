package clock

import (
	"sync"
	"time"
)

type Manual struct {
	mu  sync.RWMutex
	now time.Time
}

func NewManual(now time.Time) *Manual {
	return &Manual{
		now: now,
	}
}

func (c *Manual) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.now
}

func (c *Manual) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = now
}

func (c *Manual) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(duration)
}
