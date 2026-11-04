package serviceaccount

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	AccountKey          string
	Name                string
	Description         *string
	Status              string
	CreatedByIdentityID *uuid.UUID

	CreatedAt  time.Time
	UpdatedAt  time.Time
	DisabledAt *time.Time
	DeletedAt  *time.Time
}

type credentialRow struct {
	ID                  uuid.UUID
	ServiceAccountID    uuid.UUID
	Type                string
	SecretHash          string
	SecretPrefix        *string
	Name                *string
	CreatedByIdentityID *uuid.UUID

	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	RevokeReason *string
	LastUsedAt   *time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var m row

	if err := source.Scan(
		&m.ID,
		&m.OrganizationID,
		&m.AccountKey,
		&m.Name,
		&m.Description,
		&m.Status,
		&m.CreatedByIdentityID,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.DisabledAt,
		&m.DeletedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan service account row: %w", err)
	}

	return m, nil
}

func scanRows(rows pgx.Rows) ([]row, error) {
	m := make([]row, 0)

	for rows.Next() {
		v, err := scanRow(rows)
		if err != nil {
			return nil, err
		}

		m = append(m, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate service accounts: %w", err)
	}

	return m, nil
}

func scanCredentialRows(rows pgx.Rows) ([]credentialRow, error) {
	m := make([]credentialRow, 0)

	for rows.Next() {
		var v credentialRow

		if err := rows.Scan(
			&v.ID,
			&v.ServiceAccountID,
			&v.Type,
			&v.SecretHash,
			&v.SecretPrefix,
			&v.Name,
			&v.CreatedByIdentityID,
			&v.CreatedAt,
			&v.UpdatedAt,
			&v.ExpiresAt,
			&v.RevokedAt,
			&v.RevokeReason,
			&v.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan service account credential: %w", err)
		}

		m = append(m, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate service account credentials: %w", err)
	}

	return m, nil
}
