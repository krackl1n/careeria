package identityprofile

import (
	"context"
	"errors"
	"fmt"

	sharedpostgres "github.com/krackl1n/careeria/postgres"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
)

func (r *PostgresRepository) Create(
	ctx context.Context,
	entity *identityprofiledomain.Profile,
) error {
	if entity == nil {
		return errors.New("create identity profile: entity is nil")
	}

	_, err := r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.identity_profiles (
			identity_id,
			family_name,
			given_name,
			middle_name,
			source,
			assurance_level,
			version,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entity.IdentityID.UUID(),
		entity.FamilyName,
		entity.GivenName,
		entity.MiddleName,
		string(entity.Source),
		string(entity.AssuranceLevel),
		entity.Version,
		entity.CreatedAt,
		entity.UpdatedAt,
	)

	if err != nil {
		if sharedpostgres.IsUniqueViolation(err, "identity_profiles_pkey") {
			return identityprofiledomain.ErrAlreadyExists
		}

		return fmt.Errorf("create identity profile: %w", err)
	}

	return nil
}
