package password

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type credentialRow struct {
	IdentityID        uuid.UUID
	PasswordHash      string
	MustChange        bool
	PasswordUpdatedAt time.Time
	ExpiresAt         *time.Time
	CompromisedAt     *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type authStateRow struct {
	IdentityID     uuid.UUID
	FailedAttempts int
	LastFailedAt   *time.Time
	LockedUntil    *time.Time
	LockReason     *string
	UpdatedAt      time.Time
}

func scanCredentialRow(row pgx.Row) (credentialRow, error) {
	var model credentialRow

	if err := row.Scan(
		&model.IdentityID,
		&model.PasswordHash,
		&model.MustChange,
		&model.PasswordUpdatedAt,
		&model.ExpiresAt,
		&model.CompromisedAt,
		&model.CreatedAt,
		&model.UpdatedAt,
	); err != nil {
		return credentialRow{}, fmt.Errorf("scan password credential row: %w", err)
	}

	return model, nil
}

func scanAuthStateRow(row pgx.Row) (authStateRow, error) {
	var model authStateRow

	if err := row.Scan(
		&model.IdentityID,
		&model.FailedAttempts,
		&model.LastFailedAt,
		&model.LockedUntil,
		&model.LockReason,
		&model.UpdatedAt,
	); err != nil {
		return authStateRow{}, fmt.Errorf("scan password authentication state row: %w", err)
	}

	return model, nil
}
