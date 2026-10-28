package emailverificationchallenge

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID          uuid.UUID
	EmailID     uuid.UUID
	TokenHash   string
	Attempts    int
	MaxAttempts int

	RequestedFromIP *string
	ConsumedFromIP  *string

	CreatedAt     time.Time
	ExpiresAt     time.Time
	LastAttemptAt *time.Time
	ConsumedAt    *time.Time
	RevokedAt     *time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var model row

	if err := source.Scan(
		&model.ID,
		&model.EmailID,
		&model.TokenHash,
		&model.Attempts,
		&model.MaxAttempts,
		&model.RequestedFromIP,
		&model.ConsumedFromIP,
		&model.CreatedAt,
		&model.ExpiresAt,
		&model.LastAttemptAt,
		&model.ConsumedAt,
		&model.RevokedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan email verification challenge row: %w", err)
	}

	return model, nil
}
