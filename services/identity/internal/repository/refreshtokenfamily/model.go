package refreshtokenfamily

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID        uuid.UUID
	SessionID uuid.UUID

	CreatedAt         time.Time
	LastRotatedAt     *time.Time
	AbsoluteExpiresAt time.Time
	ReuseDetectedAt   *time.Time
	RevokedAt         *time.Time
	RevokeReason      *string
}

type tokenRow struct {
	ID            uuid.UUID
	FamilyID      uuid.UUID
	ParentTokenID *uuid.UUID
	TokenHash     string

	IssuedAt        time.Time
	ExpiresAt       time.Time
	RotatedAt       *time.Time
	RevokedAt       *time.Time
	RevokeReason    *string
	ReuseDetectedAt *time.Time

	CreatedFromIP    *string
	CreatedUserAgent *string
}

func scanRow(source pgx.Row) (row, error) {
	var m row

	if err := source.Scan(
		&m.ID,
		&m.SessionID,
		&m.CreatedAt,
		&m.LastRotatedAt,
		&m.AbsoluteExpiresAt,
		&m.ReuseDetectedAt,
		&m.RevokedAt,
		&m.RevokeReason,
	); err != nil {
		return row{}, fmt.Errorf("scan refresh token family row: %w", err)
	}

	return m, nil
}

func scanTokenRows(rows pgx.Rows) ([]tokenRow, error) {
	models := make([]tokenRow, 0)

	for rows.Next() {
		var m tokenRow

		if err := rows.Scan(
			&m.ID,
			&m.FamilyID,
			&m.ParentTokenID,
			&m.TokenHash,
			&m.IssuedAt,
			&m.ExpiresAt,
			&m.RotatedAt,
			&m.RevokedAt,
			&m.RevokeReason,
			&m.ReuseDetectedAt,
			&m.CreatedFromIP,
			&m.CreatedUserAgent,
		); err != nil {
			return nil, fmt.Errorf("scan refresh token row: %w", err)
		}

		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate refresh tokens: %w", err)
	}

	return models, nil
}
