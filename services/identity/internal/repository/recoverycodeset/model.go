package recoverycodeset

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID         uuid.UUID
	IdentityID uuid.UUID

	CreatedAt  time.Time
	ExpiresAt  *time.Time
	DisabledAt *time.Time
}

type codeRow struct {
	ID        uuid.UUID
	SetID     uuid.UUID
	CodeHash  string
	CreatedAt time.Time
	UsedAt    *time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var m row

	if err := source.Scan(
		&m.ID,
		&m.IdentityID,
		&m.CreatedAt,
		&m.ExpiresAt,
		&m.DisabledAt,
	); err != nil {
		return row{}, fmt.Errorf("scan recovery code set row: %w", err)
	}

	return m, nil
}

func scanCodeRows(rows pgx.Rows) ([]codeRow, error) {
	m := make([]codeRow, 0)

	for rows.Next() {
		var v codeRow

		if err := rows.Scan(
			&v.ID,
			&v.SetID,
			&v.CodeHash,
			&v.CreatedAt,
			&v.UsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recovery code row: %w", err)
		}

		m = append(m, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recovery codes: %w", err)
	}

	return m, nil
}
