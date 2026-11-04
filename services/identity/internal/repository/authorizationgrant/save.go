package authorizationgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	grantdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationgrant"
)

func (r *PostgresRepository) Create(ctx context.Context, grant *grantdomain.AuthorizationGrant) error {
	if grant == nil {
		return errors.New("create authorization grant: entity is nil")
	}

	batch := &pgx.Batch{}
	batch.Queue(`
		INSERT INTO public.oauth_authorization_grants (
			id,
			identity_id,
			oauth_client_id,
			source,
			granted_at,
			updated_at,
			expires_at,
			revoked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		grant.ID.UUID(),
		grant.IdentityID.UUID(),
		grant.OAuthClientID.UUID(),
		string(grant.Source),
		grant.GrantedAt,
		grant.UpdatedAt,
		grant.ExpiresAt,
		grant.RevokedAt,
	)
	queueScopes(batch, grant)

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if sharedpostgres.IsUniqueViolation(err, "oauth_authorization_grants_identity_client_uk") {
			return grantdomain.ErrAlreadyExists
		}
		return fmt.Errorf("create authorization grant: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, grant *grantdomain.AuthorizationGrant) error {
	if grant == nil {
		return errors.New("update authorization grant: entity is nil")
	}

	batch := &pgx.Batch{}
	batch.Queue(`
		UPDATE public.oauth_authorization_grants
		SET
			source = $2,
			updated_at = $3,
			expires_at = $4,
			revoked_at = $5
		WHERE id = $1`,
		grant.ID.UUID(),
		string(grant.Source),
		grant.UpdatedAt,
		grant.ExpiresAt,
		grant.RevokedAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return grantdomain.ErrNotFound
		}
		return nil
	})
	batch.Queue(`DELETE FROM public.oauth_authorization_grant_scopes WHERE grant_id = $1`, grant.ID.UUID())
	queueScopes(batch, grant)

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, grantdomain.ErrNotFound) {
			return grantdomain.ErrNotFound
		}
		return fmt.Errorf("update authorization grant: %w", err)
	}
	return nil
}

func queueScopes(batch *pgx.Batch, grant *grantdomain.AuthorizationGrant) {
	for _, scope := range grant.Scopes {
		batch.Queue(`
			INSERT INTO public.oauth_authorization_grant_scopes (
				grant_id,
				scope,
				granted_at
			)
			VALUES ($1, $2, $3)`,
			grant.ID.UUID(),
			string(scope),
			grant.UpdatedAt,
		)
	}
}
