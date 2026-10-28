package identityprofile

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	identityID identitydomain.ID,
) (*identityprofiledomain.Profile, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
		SELECT
			identity_id,
			family_name,
			given_name,
			middle_name,
			source,
			assurance_level,
			version,
			created_at,
			updated_at
		FROM public.identity_profiles
		WHERE identity_id = $1`,
		identityID.UUID(),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identityprofiledomain.ErrNotFound
		}

		return nil, fmt.Errorf("get identity profile: %w", err)
	}

	return toDomain(model), nil
}
