package authorizationgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	grantdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationgrant"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	identityID identitydomain.ID,
	oauthClientID oauthclientdomain.ID,
) (*grantdomain.AuthorizationGrant, error) {
	var grant row
	var scopes []scopeRow
	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT
			id,
			identity_id,
			oauth_client_id,
			source,
			granted_at,
			updated_at,
			expires_at,
			revoked_at
		FROM public.oauth_authorization_grants
		WHERE identity_id = $1
			AND oauth_client_id = $2`,
		identityID.UUID(),
		oauthClientID.UUID(),
	).QueryRow(func(source pgx.Row) error {
		var err error
		grant, err = scanRow(source)
		return err
	})

	batch.Queue(`
		SELECT
			s.grant_id,
			s.scope,
			s.granted_at
		FROM public.oauth_authorization_grant_scopes AS s
		JOIN public.oauth_authorization_grants AS g ON g.id = s.grant_id
		WHERE g.identity_id = $1
			AND g.oauth_client_id = $2
		ORDER BY s.granted_at, s.scope`,
		identityID.UUID(),
		oauthClientID.UUID(),
	).Query(func(rows pgx.Rows) error {
		var err error
		scopes, err = scanScopeRows(rows)
		return err
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, grantdomain.ErrNotFound
		}
		return nil, fmt.Errorf("get authorization grant: %w", err)
	}

	return toDomain(grant, scopes), nil
}
