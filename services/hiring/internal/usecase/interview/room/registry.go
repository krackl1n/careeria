package room

import (
	"errors"
	"fmt"
	"sync"

	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

var ErrUnsupportedFormat = errors.New("unsupported meeting format")

type Registry interface {
	Get(format interviewdomain.MeetingFormat) (MeetingRoomStrategy, error)
	Register(strategy MeetingRoomStrategy)
}

type inMemRegistry struct {
	mu         sync.RWMutex
	strategies map[interviewdomain.MeetingFormat]MeetingRoomStrategy
}

func NewRegistry(strategies ...MeetingRoomStrategy) Registry {
	r := &inMemRegistry{
		strategies: make(map[interviewdomain.MeetingFormat]MeetingRoomStrategy),
	}
	for _, s := range strategies {
		r.Register(s)
	}
	return r
}

func (r *inMemRegistry) Register(strategy MeetingRoomStrategy) {
	if strategy == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[strategy.Format()] = strategy
}

func (r *inMemRegistry) Get(format interviewdomain.MeetingFormat) (MeetingRoomStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.strategies[format]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return s, nil
}

