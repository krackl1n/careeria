package identityprofile

import (
	"context"
	"errors"
	"fmt"

	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
)

func (r *PostgresRepository) Update(
	ctx context.Context,
	entity *identityprofiledomain.Profile,
) error {
	if entity == nil {
		return errors.New("update identity profile: entity is nil")
	}

	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.identity_profiles
		SET
			family_name = $2,
			given_name = $3,
			middle_name = $4,
			source = $5,
			assurance_level = $6,
			version = $7,
			updated_at = $8
		WHERE identity_id = $1
			AND version = $7 - 1`,
		entity.IdentityID.UUID(),
		entity.FamilyName,
		entity.GivenName,
		entity.MiddleName,
		string(entity.Source),
		string(entity.AssuranceLevel),
		entity.Version,
		entity.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update identity profile: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return identityprofiledomain.ErrNotFound
	}

	return nil
}
