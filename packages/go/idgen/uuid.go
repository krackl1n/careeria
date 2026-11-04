package idgen

import "github.com/google/uuid"

type UUID struct{}

func NewUUID() UUID {
	return UUID{}
}

func (UUID) New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

func (g UUID) Generate() string {
	return g.New().String()
}
