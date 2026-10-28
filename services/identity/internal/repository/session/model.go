package session

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID            uuid.UUID
	IdentityID    uuid.UUID
	OAuthClientID uuid.UUID

	AuthenticationLevel  string
	AuthenticatedMethods []string
	Scopes               []string

	IPAddress  *string
	UserAgent  *string
	DeviceName *string

	CreatedAt         time.Time
	AuthenticatedAt   time.Time
	LastSeenAt        *time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time

	RevokedAt    *time.Time
	RevokeReason *string
}

func scanRow(source pgx.Row) (row, error) {
	var model row

	if err := source.Scan(
		&model.ID,
		&model.IdentityID,
		&model.OAuthClientID,
		&model.AuthenticationLevel,
		&model.AuthenticatedMethods,
		&model.Scopes,
		&model.IPAddress,
		&model.UserAgent,
		&model.DeviceName,
		&model.CreatedAt,
		&model.AuthenticatedAt,
		&model.LastSeenAt,
		&model.IdleExpiresAt,
		&model.AbsoluteExpiresAt,
		&model.RevokedAt,
		&model.RevokeReason,
	); err != nil {
		return row{}, fmt.Errorf("scan session row: %w", err)
	}

	return model, nil
}

func scanRows(rows pgx.Rows) ([]row, error) {
	models := make([]row, 0)

	for rows.Next() {
		model, err := scanRow(rows)
		if err != nil {
			return nil, err
		}

		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return models, nil
}
