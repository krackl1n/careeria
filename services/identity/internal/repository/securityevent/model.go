package securityevent

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID        uuid.UUID
	EventType string
	Result    string
	ActorType string

	IdentityID       *uuid.UUID
	ServiceAccountID *uuid.UUID
	SessionID        *uuid.UUID
	OAuthClientID    *uuid.UUID

	IdentifierHash []byte
	IPAddress      *string
	UserAgent      *string
	CorrelationID  *uuid.UUID

	OccurredAt time.Time
	Metadata   []byte
}

func scanRow(source pgx.Row) (row, error) {
	var m row

	if err := source.Scan(
		&m.ID,
		&m.EventType,
		&m.Result,
		&m.ActorType,
		&m.IdentityID,
		&m.ServiceAccountID,
		&m.SessionID,
		&m.OAuthClientID,
		&m.IdentifierHash,
		&m.IPAddress,
		&m.UserAgent,
		&m.CorrelationID,
		&m.OccurredAt,
		&m.Metadata,
	); err != nil {
		return row{}, fmt.Errorf("scan security event row: %w", err)
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
		return nil, fmt.Errorf("iterate security events: %w", err)
	}

	return m, nil
}
