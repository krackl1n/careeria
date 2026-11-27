package vacancy

import (
	"time"

	"github.com/google/uuid"
)

type Cursor struct {
	Time time.Time
	ID   uuid.UUID
}

type Page struct {
	Vacancies []*Vacancy
	Next      *Cursor
}
