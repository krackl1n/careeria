package refreshtokenfamily

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
)

func (r *PostgresRepository) Update(ctx context.Context, e *domain.RefreshTokenFamily) error {
	if e == nil {
		return errors.New("update refresh token family: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		UPDATE public.refresh_token_families
		SET
			last_rotated_at = $2,
			reuse_detected_at = $3,
			revoked_at = $4,
			revoke_reason = $5
		WHERE id = $1`,
		e.ID.UUID(),
		e.LastRotatedAt,
		e.ReuseDetectedAt,
		e.RevokedAt,
		e.RevokeReason,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}

		return nil
	})

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
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
			WHERE EXISTS (
				SELECT 1
				FROM public.refresh_token_families
				WHERE id = $2
			)
			ON CONFLICT (id) DO UPDATE
			SET
				rotated_at = EXCLUDED.rotated_at,
				revoked_at = EXCLUDED.revoked_at,
				revoke_reason = EXCLUDED.revoke_reason,
				reuse_detected_at = EXCLUDED.reuse_detected_at`,
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
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}

		return fmt.Errorf("update refresh token family: %w", err)
	}

	return nil
}
