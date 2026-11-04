package mfa

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	mfadomain "github.com/krackl1n/careeria/services/identity/internal/domain/mfa"
)

func (r *PostgresRepository) Create(ctx context.Context, e *mfadomain.MFA) error {
	return r.save(ctx, e, false)
}

func (r *PostgresRepository) Update(ctx context.Context, e *mfadomain.MFA) error {
	return r.save(ctx, e, true)
}

func (r *PostgresRepository) save(ctx context.Context, e *mfadomain.MFA, upsert bool) error {
	if e == nil {
		return errors.New("save MFA configuration: entity is nil")
	}

	b := &pgx.Batch{}

	for _, f := range e.Factors {
		if upsert {
			b.Queue(`
				INSERT INTO public.mfa_factors (
					id,
					identity_id,
					factor_type,
					name,
					is_primary,
					confirmed_at,
					created_at,
					updated_at,
					last_used_at,
					disabled_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (id) DO UPDATE
				SET
					name = EXCLUDED.name,
					is_primary = EXCLUDED.is_primary,
					confirmed_at = EXCLUDED.confirmed_at,
					updated_at = EXCLUDED.updated_at,
					last_used_at = EXCLUDED.last_used_at,
					disabled_at = EXCLUDED.disabled_at`,
				f.ID.UUID(),
				e.IdentityID.UUID(),
				string(f.Type),
				f.Name,
				f.IsPrimary,
				f.ConfirmedAt,
				f.CreatedAt,
				f.UpdatedAt,
				f.LastUsedAt,
				f.DisabledAt,
			)
		} else {
			b.Queue(`
				INSERT INTO public.mfa_factors (
					id,
					identity_id,
					factor_type,
					name,
					is_primary,
					confirmed_at,
					created_at,
					updated_at,
					last_used_at,
					disabled_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				f.ID.UUID(),
				e.IdentityID.UUID(),
				string(f.Type),
				f.Name,
				f.IsPrimary,
				f.ConfirmedAt,
				f.CreatedAt,
				f.UpdatedAt,
				f.LastUsedAt,
				f.DisabledAt,
			)
		}

		if f.TOTP != nil {
			t := f.TOTP

			b.Queue(`
				INSERT INTO public.totp_factors (
					factor_id,
					factor_type,
					secret_encrypted,
					encryption_key_ref,
					algorithm,
					digits,
					period_seconds,
					last_used_time_step,
					updated_at
				)
				VALUES ($1, 'totp', $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (factor_id) DO UPDATE
				SET
					secret_encrypted = EXCLUDED.secret_encrypted,
					encryption_key_ref = EXCLUDED.encryption_key_ref,
					algorithm = EXCLUDED.algorithm,
					digits = EXCLUDED.digits,
					period_seconds = EXCLUDED.period_seconds,
					last_used_time_step = EXCLUDED.last_used_time_step,
					updated_at = EXCLUDED.updated_at`,
				f.ID.UUID(),
				[]byte(t.SecretEncrypted),
				string(t.EncryptionKeyRef),
				string(t.Algorithm),
				int16(t.Digits),
				int16(t.PeriodSeconds),
				t.LastUsedTimeStep,
				t.UpdatedAt,
			)
		}

		if f.WebAuthn != nil {
			w := f.WebAuthn

			b.Queue(`
				INSERT INTO public.webauthn_credentials (
					factor_id,
					factor_type,
					credential_id,
					public_key,
					sign_count,
					aaguid,
					transports,
					backup_eligible,
					backup_state,
					created_at,
					last_used_at
				)
				VALUES ($1, 'webauthn', $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (factor_id) DO UPDATE
				SET
					public_key = EXCLUDED.public_key,
					sign_count = EXCLUDED.sign_count,
					aaguid = EXCLUDED.aaguid,
					transports = EXCLUDED.transports,
					backup_eligible = EXCLUDED.backup_eligible,
					backup_state = EXCLUDED.backup_state,
					last_used_at = EXCLUDED.last_used_at`,
				f.ID.UUID(),
				[]byte(w.CredentialID),
				[]byte(w.PublicKey),
				w.SignCount,
				w.AAGUID,
				transportsToRow(w.Transports),
				w.BackupEligible,
				w.BackupState,
				w.CreatedAt,
				w.LastUsedAt,
			)
		}
	}

	if b.Len() == 0 {
		return nil
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if sharedpostgres.IsUniqueViolation(err, "mfa_factors_enabled_totp_uidx") {
			return mfadomain.ErrEnabledTOTPAlreadyExists
		}
		if sharedpostgres.IsUniqueViolation(err, "webauthn_credentials_credential_id_uidx") {
			return mfadomain.ErrCredentialAlreadyExists
		}
		if sharedpostgres.IsUniqueViolation(err, "mfa_factors_pkey") {
			return mfadomain.ErrFactorAlreadyExists
		}

		return fmt.Errorf("save MFA configuration: %w", err)
	}

	return nil
}
