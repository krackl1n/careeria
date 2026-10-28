package closer

import (
	"errors"
	"io"
	"sync"
)

type Func func() error

func (f Func) Close() error {
	if f == nil {
		return nil
	}

	return f()
}

type Closer struct {
	mu      sync.Mutex
	closed  bool
	closers []io.Closer
}

func New() *Closer {
	return &Closer{}
}

func (c *Closer) Add(closer io.Closer) {
	if closer == nil {
		return
	}

	c.mu.Lock()

	if !c.closed {
		c.closers = append(c.closers, closer)
		c.mu.Unlock()

		return
	}

	c.mu.Unlock()

	_ = closer.Close()
}

func (c *Closer) AddFunc(fn func() error) {
	if fn == nil {
		return
	}

	c.Add(Func(fn))
}

func (c *Closer) Close() error {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()

		return nil
	}

	c.closed = true

	closers := c.closers
	c.closers = nil

	c.mu.Unlock()

	errs := make([]error, 0)

	for i := len(closers) - 1; i >= 0; i-- {
		if err := closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
