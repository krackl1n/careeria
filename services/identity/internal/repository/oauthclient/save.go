package oauthclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (r *PostgresRepository) Create(ctx context.Context, e *oauthdomain.OAuthClient) error {
	if e == nil {
		return errors.New("create OAuth client: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		INSERT INTO public.oauth_clients (
			id,
			client_id,
			client_type,
			token_endpoint_auth_method,
			name,
			description,
			require_pkce,
			consent_mode,
			status,
			created_by_identity_id,
			created_at,
			updated_at,
			disabled_at,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		e.ID.UUID(),
		string(e.ClientID),
		string(e.ClientType),
		string(e.TokenEndpointAuthMethod),
		e.Name,
		e.Description,
		e.RequirePKCE,
		string(e.ConsentMode),
		string(e.Status),
		identityIDToRow(e.CreatedByIdentityID),
		e.CreatedAt,
		e.UpdatedAt,
		e.DisabledAt,
		e.DeletedAt,
	)

	queueChildren(b, e)

	return r.finishSave(ctx, b, "create")
}

func (r *PostgresRepository) Update(ctx context.Context, e *oauthdomain.OAuthClient) error {
	if e == nil {
		return errors.New("update OAuth client: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		UPDATE public.oauth_clients
		SET
			name = $2,
			description = $3,
			require_pkce = $4,
			consent_mode = $5,
			status = $6,
			updated_at = $7,
			disabled_at = $8,
			deleted_at = $9
		WHERE id = $1`,
		e.ID.UUID(),
		e.Name,
		e.Description,
		e.RequirePKCE,
		string(e.ConsentMode),
		string(e.Status),
		e.UpdatedAt,
		e.DisabledAt,
		e.DeletedAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return oauthdomain.ErrNotFound
		}

		return nil
	})

	b.Queue(`
		DELETE FROM public.oauth_client_redirect_uris
		WHERE oauth_client_id = $1`,
		e.ID.UUID(),
	)

	b.Queue(`
		DELETE FROM public.oauth_client_post_logout_redirect_uris
		WHERE oauth_client_id = $1`,
		e.ID.UUID(),
	)

	b.Queue(`
		DELETE FROM public.oauth_client_grant_types
		WHERE oauth_client_id = $1`,
		e.ID.UUID(),
	)

	b.Queue(`
		DELETE FROM public.oauth_client_scopes
		WHERE oauth_client_id = $1`,
		e.ID.UUID(),
	)

	queueChildren(b, e)

	return r.finishSave(ctx, b, "update")
}

func queueChildren(b *pgx.Batch, e *oauthdomain.OAuthClient) {
	clientID := e.ID.UUID()

	for _, v := range e.RedirectURIs {
		b.Queue(`
			INSERT INTO public.oauth_client_redirect_uris (
				id,
				oauth_client_id,
				redirect_uri,
				created_at
			)
			VALUES ($1, $2, $3, $4)`,
			v.ID.UUID(),
			clientID,
			string(v.URI),
			v.CreatedAt,
		)
	}

	for _, v := range e.PostLogoutRedirectURIs {
		b.Queue(`
			INSERT INTO public.oauth_client_post_logout_redirect_uris (
				id,
				oauth_client_id,
				post_logout_redirect_uri,
				created_at
			)
			VALUES ($1, $2, $3, $4)`,
			v.ID.UUID(),
			clientID,
			string(v.URI),
			v.CreatedAt,
		)
	}

	for _, v := range e.GrantTypes {
		b.Queue(`
			INSERT INTO public.oauth_client_grant_types (
				oauth_client_id,
				grant_type,
				created_at
			)
			VALUES ($1, $2, $3)`,
			clientID,
			string(v.GrantType),
			v.CreatedAt,
		)
	}

	for _, v := range e.Scopes {
		b.Queue(`
			INSERT INTO public.oauth_client_scopes (
				oauth_client_id,
				scope,
				created_at
			)
			VALUES ($1, $2, $3)`,
			clientID,
			string(v.Scope),
			v.CreatedAt,
		)
	}

	for _, v := range e.Credentials {
		b.Queue(`
			INSERT INTO public.oauth_client_credentials (
				id,
				oauth_client_id,
				client_type,
				secret_hash,
				name,
				created_by_identity_id,
				created_at,
				updated_at,
				expires_at,
				revoked_at,
				revoke_reason,
				last_used_at
			)
			VALUES ($1, $2, 'confidential', $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE
			SET
				name = EXCLUDED.name,
				updated_at = EXCLUDED.updated_at,
				expires_at = EXCLUDED.expires_at,
				revoked_at = EXCLUDED.revoked_at,
				revoke_reason = EXCLUDED.revoke_reason,
				last_used_at = EXCLUDED.last_used_at`,
			v.ID.UUID(),
			clientID,
			string(v.SecretHash),
			v.Name,
			identityIDToRow(v.CreatedByIdentityID),
			v.CreatedAt,
			v.UpdatedAt,
			v.ExpiresAt,
			v.RevokedAt,
			v.RevokeReason,
			v.LastUsedAt,
		)
	}
}

func (r *PostgresRepository) finishSave(ctx context.Context, b *pgx.Batch, op string) error {
	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, oauthdomain.ErrNotFound) {
			return oauthdomain.ErrNotFound
		}

		if sharedpostgres.IsUniqueViolation(err, "oauth_clients_client_id_uidx") {
			return oauthdomain.ErrClientIDAlreadyExists
		}

		if sharedpostgres.IsUniqueViolation(err, "oauth_client_redirect_uris_client_uri_uk") {
			return oauthdomain.ErrRedirectURIAlreadyExists
		}

		if sharedpostgres.IsUniqueViolation(err, "oauth_client_post_logout_redirect_uris_client_uri_uk") {
			return oauthdomain.ErrPostLogoutRedirectURIAlreadyExists
		}

		if sharedpostgres.IsUniqueViolation(err, "oauth_client_grant_types_pk") {
			return oauthdomain.ErrGrantTypeAlreadyExists
		}

		if sharedpostgres.IsUniqueViolation(err, "oauth_client_scopes_pk") {
			return oauthdomain.ErrScopeAlreadyExists
		}

		return fmt.Errorf("%s OAuth client: %w", op, err)
	}

	return nil
}
