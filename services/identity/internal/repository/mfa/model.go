package mfa

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type factorRow struct {
	ID          uuid.UUID
	IdentityID  uuid.UUID
	FactorType  string
	Name        *string
	IsPrimary   bool
	ConfirmedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastUsedAt  *time.Time
	DisabledAt  *time.Time

	TOTPSecretEncrypted  []byte
	TOTPEncryptionKeyRef *string
	TOTPAlgorithm        *string
	TOTPDigits           *int16
	TOTPPeriodSeconds    *int16
	TOTPLastUsedTimeStep *int64
	TOTPUpdatedAt        *time.Time

	WebAuthnCredentialID   []byte
	WebAuthnPublicKey      []byte
	WebAuthnSignCount      *int64
	WebAuthnAAGUID         *uuid.UUID
	WebAuthnTransports     []string
	WebAuthnBackupEligible *bool
	WebAuthnBackupState    *bool
	WebAuthnCreatedAt      *time.Time
	WebAuthnLastUsedAt     *time.Time
}

func scanFactorRows(rows pgx.Rows) ([]factorRow, error) {
	models := make([]factorRow, 0)

	for rows.Next() {
		var m factorRow

		if err := rows.Scan(
			&m.ID,
			&m.IdentityID,
			&m.FactorType,
			&m.Name,
			&m.IsPrimary,
			&m.ConfirmedAt,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.LastUsedAt,
			&m.DisabledAt,
			&m.TOTPSecretEncrypted,
			&m.TOTPEncryptionKeyRef,
			&m.TOTPAlgorithm,
			&m.TOTPDigits,
			&m.TOTPPeriodSeconds,
			&m.TOTPLastUsedTimeStep,
			&m.TOTPUpdatedAt,
			&m.WebAuthnCredentialID,
			&m.WebAuthnPublicKey,
			&m.WebAuthnSignCount,
			&m.WebAuthnAAGUID,
			&m.WebAuthnTransports,
			&m.WebAuthnBackupEligible,
			&m.WebAuthnBackupState,
			&m.WebAuthnCreatedAt,
			&m.WebAuthnLastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan MFA factor row: %w", err)
		}

		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MFA factors: %w", err)
	}

	return models, nil
}
