-- +goose Up
BEGIN;

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE public.oauth_consent_mode AS ENUM (
        'required',
        'remember',
        'trusted'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE public.oauth_authorization_grant_source AS ENUM (
        'user_consent',
        'trusted_client_policy'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END;
$$;
-- +goose StatementEnd

ALTER TABLE public.oauth_clients
    ADD COLUMN IF NOT EXISTS consent_mode public.oauth_consent_mode NOT NULL DEFAULT 'remember';

CREATE TABLE IF NOT EXISTS public.oauth_authorization_grants (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE CASCADE,
    source public.oauth_authorization_grant_source NOT NULL,
    granted_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz,
    revoked_at timestamptz,

    CONSTRAINT oauth_authorization_grants_identity_client_uk
        UNIQUE (identity_id, oauth_client_id),
    CONSTRAINT oauth_authorization_grants_updated_at_chk
        CHECK (updated_at >= granted_at),
    CONSTRAINT oauth_authorization_grants_expires_at_chk
        CHECK (expires_at IS NULL OR expires_at > granted_at),
    CONSTRAINT oauth_authorization_grants_revoked_at_chk
        CHECK (revoked_at IS NULL OR revoked_at >= granted_at)
);

CREATE TABLE IF NOT EXISTS public.oauth_authorization_grant_scopes (
    grant_id uuid NOT NULL
        REFERENCES public.oauth_authorization_grants (id) ON DELETE CASCADE,
    scope text NOT NULL,
    granted_at timestamptz NOT NULL,

    CONSTRAINT oauth_authorization_grant_scopes_scope_chk
        CHECK (
            scope = btrim(scope)
            AND scope <> ''
            AND scope !~ '[[:space:]]'
        ),
    CONSTRAINT oauth_authorization_grant_scopes_pk
        PRIMARY KEY (grant_id, scope)
);

CREATE INDEX IF NOT EXISTS oauth_authorization_grants_identity_idx
    ON public.oauth_authorization_grants (identity_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS oauth_authorization_grants_client_idx
    ON public.oauth_authorization_grants (oauth_client_id)
    WHERE revoked_at IS NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.oauth_authorization_grant_scopes;
DROP TABLE IF EXISTS public.oauth_authorization_grants;
ALTER TABLE public.oauth_clients DROP COLUMN IF EXISTS consent_mode;
DROP TYPE IF EXISTS public.oauth_authorization_grant_source;
DROP TYPE IF EXISTS public.oauth_consent_mode;

COMMIT;
