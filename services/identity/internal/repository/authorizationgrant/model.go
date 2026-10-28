package authorizationgrant

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
	Source        string
	GrantedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     *time.Time
	RevokedAt     *time.Time
}

type scopeRow struct {
	GrantID   uuid.UUID
	Scope     string
	GrantedAt time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var result row
	if err := source.Scan(
		&result.ID,
		&result.IdentityID,
		&result.OAuthClientID,
		&result.Source,
		&result.GrantedAt,
		&result.UpdatedAt,
		&result.ExpiresAt,
		&result.RevokedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan authorization grant row: %w", err)
	}
	return result, nil
}

func scanScopeRows(rows pgx.Rows) ([]scopeRow, error) {
	result := make([]scopeRow, 0)
	for rows.Next() {
		var value scopeRow
		if err := rows.Scan(&value.GrantID, &value.Scope, &value.GrantedAt); err != nil {
			return nil, fmt.Errorf("scan authorization grant scope row: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authorization grant scopes: %w", err)
	}
	return result, nil
}
