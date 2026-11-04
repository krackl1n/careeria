package idgen

import "sync"

type Sequence struct {
	mu     sync.Mutex
	values []string
	index  int
}

func NewSequence(values ...string) *Sequence {
	copied := make([]string, len(values))
	copy(copied, values)

	for _, value := range copied {
		if value == "" {
			panic("idgen: empty sequence value")
		}
	}

	return &Sequence{
		values: copied,
	}
}

func (s *Sequence) Generate() string {
	if s == nil {
		panic("idgen: nil sequence")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index >= len(s.values) {
		panic("idgen: sequence exhausted")
	}

	value := s.values[s.index]
	s.index++

	return value
}

func (s *Sequence) Add(values ...string) {
	if s == nil {
		panic("idgen: nil sequence")
	}

	for _, value := range values {
		if value == "" {
			panic("idgen: empty sequence value")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.values = append(s.values, values...)
}

func (s *Sequence) Remaining() int {
	if s == nil {
		panic("idgen: nil sequence")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.values) - s.index
}
