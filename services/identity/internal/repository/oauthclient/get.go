package oauthclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id oauthdomain.ID,
) (*oauthdomain.OAuthClient, error) {
	value := id.UUID()

	return r.get(ctx, &value, nil)
}

func (r *PostgresRepository) GetByClientID(
	ctx context.Context,
	id oauthdomain.ClientID,
) (*oauthdomain.OAuthClient, error) {
	value := string(id)

	return r.get(ctx, nil, &value)
}

func (r *PostgresRepository) get(
	ctx context.Context,
	id any,
	clientID *string,
) (*oauthdomain.OAuthClient, error) {
	var m row
	var redirects []redirectURIRow
	var postLogoutRedirects []redirectURIRow
	var grantTypes []registrationRow
	var scopes []registrationRow
	var credentials []credentialRow
	b := &pgx.Batch{}

	b.Queue(`
		SELECT
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
		FROM public.oauth_clients
		WHERE ($1::uuid IS NOT NULL AND id = $1)
			OR ($2::text IS NOT NULL AND client_id = $2)`,
		id,
		clientID,
	).QueryRow(func(row pgx.Row) error {
		v, err := scanRow(row)
		if err != nil {
			return err
		}

		m = v
		return nil
	})

	b.Queue(`
		SELECT
			r.id,
			r.oauth_client_id,
			r.redirect_uri,
			r.created_at
		FROM public.oauth_client_redirect_uris AS r
		JOIN public.oauth_clients AS c ON c.id = r.oauth_client_id
		WHERE ($1::uuid IS NOT NULL AND c.id = $1)
			OR ($2::text IS NOT NULL AND c.client_id = $2)
		ORDER BY r.created_at, r.id`,
		id,
		clientID,
	).Query(func(rows pgx.Rows) error {
		var err error
		redirects, err = scanRedirectRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	b.Queue(`
		SELECT
			r.id,
			r.oauth_client_id,
			r.post_logout_redirect_uri,
			r.created_at
		FROM public.oauth_client_post_logout_redirect_uris AS r
		JOIN public.oauth_clients AS c ON c.id = r.oauth_client_id
		WHERE ($1::uuid IS NOT NULL AND c.id = $1)
			OR ($2::text IS NOT NULL AND c.client_id = $2)
		ORDER BY r.created_at, r.id`,
		id,
		clientID,
	).Query(func(rows pgx.Rows) error {
		var err error
		postLogoutRedirects, err = scanRedirectRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	b.Queue(`
		SELECT
			r.oauth_client_id,
			r.grant_type,
			r.created_at
		FROM public.oauth_client_grant_types AS r
		JOIN public.oauth_clients AS c ON c.id = r.oauth_client_id
		WHERE ($1::uuid IS NOT NULL AND c.id = $1)
			OR ($2::text IS NOT NULL AND c.client_id = $2)
		ORDER BY r.created_at, r.grant_type`,
		id,
		clientID,
	).Query(func(rows pgx.Rows) error {
		var err error
		grantTypes, err = scanRegistrationRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	b.Queue(`
		SELECT
			r.oauth_client_id,
			r.scope,
			r.created_at
		FROM public.oauth_client_scopes AS r
		JOIN public.oauth_clients AS c ON c.id = r.oauth_client_id
		WHERE ($1::uuid IS NOT NULL AND c.id = $1)
			OR ($2::text IS NOT NULL AND c.client_id = $2)
		ORDER BY r.created_at, r.scope`,
		id,
		clientID,
	).Query(func(rows pgx.Rows) error {
		var err error
		scopes, err = scanRegistrationRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	b.Queue(`
		SELECT
			r.id,
			r.oauth_client_id,
			r.secret_hash,
			r.name,
			r.created_by_identity_id,
			r.created_at,
			r.updated_at,
			r.expires_at,
			r.revoked_at,
			r.revoke_reason,
			r.last_used_at
		FROM public.oauth_client_credentials AS r
		JOIN public.oauth_clients AS c ON c.id = r.oauth_client_id
		WHERE ($1::uuid IS NOT NULL AND c.id = $1)
			OR ($2::text IS NOT NULL AND c.client_id = $2)
		ORDER BY r.created_at, r.id`,
		id,
		clientID,
	).Query(func(rows pgx.Rows) error {
		var err error
		credentials, err = scanCredentialRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, oauthdomain.ErrNotFound
		}

		return nil, fmt.Errorf("get OAuth client: %w", err)
	}

	return toDomain(m, redirects, postLogoutRedirects, grantTypes, scopes, credentials), nil
}
