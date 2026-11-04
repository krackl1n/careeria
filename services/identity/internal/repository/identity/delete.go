package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) Delete(ctx context.Context, entity *identitydomain.Identity) error {
	if entity == nil || entity.Status != identitydomain.StatusDeleted || entity.DeletedAt == nil {
		return errors.New("delete identity: deleted identity state is required")
	}

	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.identities
		SET
			status = $2,
			updated_at = $3,
			deleted_at = $4
		WHERE id = $1
			AND deleted_at IS NULL`,
		entity.ID.UUID(),
		string(entity.Status),
		entity.UpdatedAt,
		entity.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("delete identity: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return identitydomain.ErrNotFound
	}

	return nil
}
