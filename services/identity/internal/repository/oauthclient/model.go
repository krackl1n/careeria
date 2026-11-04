package oauthclient

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID                      uuid.UUID
	ClientID                string
	ClientType              string
	TokenEndpointAuthMethod string
	Name                    string
	Description             *string
	RequirePKCE             bool
	ConsentMode             string
	Status                  string
	CreatedByIdentityID     *uuid.UUID

	CreatedAt  time.Time
	UpdatedAt  time.Time
	DisabledAt *time.Time
	DeletedAt  *time.Time
}

type redirectURIRow struct {
	ID            uuid.UUID
	OAuthClientID uuid.UUID
	URI           string
	CreatedAt     time.Time
}

type registrationRow struct {
	OAuthClientID uuid.UUID
	Value         string
	CreatedAt     time.Time
}

type credentialRow struct {
	ID                  uuid.UUID
	OAuthClientID       uuid.UUID
	SecretHash          string
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
		&m.ClientID,
		&m.ClientType,
		&m.TokenEndpointAuthMethod,
		&m.Name,
		&m.Description,
		&m.RequirePKCE,
		&m.ConsentMode,
		&m.Status,
		&m.CreatedByIdentityID,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.DisabledAt,
		&m.DeletedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan OAuth client row: %w", err)
	}

	return m, nil
}

func scanRedirectRows(rows pgx.Rows) ([]redirectURIRow, error) {
	m := make([]redirectURIRow, 0)

	for rows.Next() {
		var v redirectURIRow

		if err := rows.Scan(
			&v.ID,
			&v.OAuthClientID,
			&v.URI,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan OAuth client redirect URI: %w", err)
		}

		m = append(m, v)
	}

	return m, rows.Err()
}

func scanRegistrationRows(rows pgx.Rows) ([]registrationRow, error) {
	m := make([]registrationRow, 0)

	for rows.Next() {
		var v registrationRow

		if err := rows.Scan(
			&v.OAuthClientID,
			&v.Value,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan OAuth client registration: %w", err)
		}

		m = append(m, v)
	}

	return m, rows.Err()
}

func scanCredentialRows(rows pgx.Rows) ([]credentialRow, error) {
	m := make([]credentialRow, 0)

	for rows.Next() {
		var v credentialRow

		if err := rows.Scan(
			&v.ID,
			&v.OAuthClientID,
			&v.SecretHash,
			&v.Name,
			&v.CreatedByIdentityID,
			&v.CreatedAt,
			&v.UpdatedAt,
			&v.ExpiresAt,
			&v.RevokedAt,
			&v.RevokeReason,
			&v.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan OAuth client credential: %w", err)
		}

		m = append(m, v)
	}

	return m, rows.Err()
}
