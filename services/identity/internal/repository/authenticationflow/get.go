package authenticationflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (r *PostgresReader) Get(
	ctx context.Context,
	identityID identitydomain.ID,
	oauthClientID oauthclientdomain.ID,
	requestedScopes []oauthdomain.Scope,
	now time.Time,
) (flowdomain.Facts, error) {
	if !identityID.IsValid() || !oauthClientID.IsValid() {
		return flowdomain.Facts{}, errors.New("get authentication flow facts: identifier is invalid")
	}
	if len(requestedScopes) == 0 {
		return flowdomain.Facts{}, errors.New("get authentication flow facts: requested scopes are required")
	}
	if now.IsZero() {
		return flowdomain.Facts{}, errors.New("get authentication flow facts: time is invalid")
	}

	scopes := make([]string, 0, len(requestedScopes))
	seen := make(map[oauthdomain.Scope]struct{}, len(requestedScopes))
	for _, scope := range requestedScopes {
		if scope == "" {
			return flowdomain.Facts{}, errors.New("get authentication flow facts: requested scope is empty")
		}
		if _, exists := seen[scope]; exists {
			return flowdomain.Facts{}, errors.New("get authentication flow facts: requested scope is duplicated")
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, string(scope))
	}

	var availableMethods []string
	var consentMode string
	var rememberedGrantCovers bool
	err := r.client.Executor(ctx).QueryRow(ctx, `
		WITH usable_mfa_methods AS (
			SELECT DISTINCT factor_type::text AS method
			FROM public.mfa_factors
			WHERE identity_id = $1
				AND confirmed_at IS NOT NULL
				AND disabled_at IS NULL
		), available_mfa_methods AS (
			SELECT method
			FROM usable_mfa_methods

			UNION

			SELECT 'recovery_code'
			WHERE EXISTS (SELECT 1 FROM usable_mfa_methods)
				AND EXISTS (
					SELECT 1
					FROM public.recovery_code_sets AS sets
					JOIN public.recovery_codes AS codes ON codes.set_id = sets.id
					WHERE sets.identity_id = $1
						AND sets.disabled_at IS NULL
						AND (sets.expires_at IS NULL OR sets.expires_at > $4)
						AND codes.used_at IS NULL
				)
		)
		SELECT
			ARRAY(SELECT method FROM available_mfa_methods ORDER BY method),
			clients.consent_mode::text,
			CASE
				WHEN clients.consent_mode = 'remember' THEN EXISTS (
					SELECT 1
					FROM public.oauth_authorization_grants AS grants
					WHERE grants.identity_id = $1
						AND grants.oauth_client_id = $2
						AND grants.revoked_at IS NULL
						AND (grants.expires_at IS NULL OR grants.expires_at > $4)
						AND NOT EXISTS (
							SELECT 1
							FROM unnest($3::text[]) AS requested(scope)
							WHERE NOT EXISTS (
								SELECT 1
								FROM public.oauth_authorization_grant_scopes AS granted
								WHERE granted.grant_id = grants.id
									AND granted.scope = requested.scope
							)
						)
				)
				ELSE false
			END
		FROM public.oauth_clients AS clients
		WHERE clients.id = $2`,
		identityID.UUID(),
		oauthClientID.UUID(),
		scopes,
		now,
	).Scan(&availableMethods, &consentMode, &rememberedGrantCovers)
	if errors.Is(err, pgx.ErrNoRows) {
		return flowdomain.Facts{}, flowdomain.ErrFactsNotFound
	}
	if err != nil {
		return flowdomain.Facts{}, fmt.Errorf("get authentication flow facts: %w", err)
	}

	methods := make([]authenticationdomain.Method, 0, len(availableMethods))
	for _, method := range availableMethods {
		methods = append(methods, authenticationdomain.Method(method))
	}
	facts, err := flowdomain.NewFacts(
		methods,
		oauthclientdomain.ConsentMode(consentMode),
		rememberedGrantCovers,
	)
	if err != nil {
		return flowdomain.Facts{}, fmt.Errorf("convert authentication flow facts: %w", err)
	}
	return facts, nil
}
