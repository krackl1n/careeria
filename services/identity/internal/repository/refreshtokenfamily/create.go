package refreshtokenfamily

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
)

func (r *PostgresRepository) Create(ctx context.Context, e *domain.RefreshTokenFamily) error {
	if e == nil {
		return errors.New("create refresh token family: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		INSERT INTO public.refresh_token_families (
			id,
			session_id,
			created_at,
			last_rotated_at,
			absolute_expires_at,
			reuse_detected_at,
			revoked_at,
			revoke_reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ID.UUID(),
		e.SessionID.UUID(),
		e.CreatedAt,
		e.LastRotatedAt,
		e.AbsoluteExpiresAt,
		e.ReuseDetectedAt,
		e.RevokedAt,
		e.RevokeReason,
	)

	for _, token := range e.Tokens {
		t := tokenToRow(e.ID, token)
		b.Queue(`
			INSERT INTO public.refresh_tokens (
				id,
				family_id,
				parent_token_id,
				token_hash,
				issued_at,
				expires_at,
				rotated_at,
				revoked_at,
				revoke_reason,
				reuse_detected_at,
				created_from_ip,
				created_user_agent
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			t.ID,
			t.FamilyID,
			t.ParentTokenID,
			t.TokenHash,
			t.IssuedAt,
			t.ExpiresAt,
			t.RotatedAt,
			t.RevokedAt,
			t.RevokeReason,
			t.ReuseDetectedAt,
			t.CreatedFromIP,
			t.CreatedUserAgent,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if sharedpostgres.IsUniqueViolation(err, "refresh_token_families_session_id_key") {
			return domain.ErrSessionFamilyAlreadyExists
		}

		return fmt.Errorf("create refresh token family: %w", err)
	}

	return nil
}
