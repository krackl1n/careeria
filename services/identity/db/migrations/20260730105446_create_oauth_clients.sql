-- +goose Up
BEGIN;

CREATE TYPE public.oauth_client_type AS ENUM (
    'public',
    'confidential'
);

CREATE TYPE public.oauth_token_endpoint_auth_method AS ENUM (
    'none',
    'client_secret_basic'
);

CREATE TYPE public.oauth_grant_type AS ENUM (
    'authorization_code',
    'refresh_token'
);

CREATE TYPE public.oauth_client_status AS ENUM (
    'active',
    'disabled',
    'deleted'
);

CREATE TABLE IF NOT EXISTS public.oauth_clients (
    id uuid PRIMARY KEY,
    client_id text NOT NULL,
    client_type public.oauth_client_type NOT NULL,
    token_endpoint_auth_method public.oauth_token_endpoint_auth_method NOT NULL,
    name text NOT NULL,
    description text,
    require_pkce boolean NOT NULL DEFAULT true,
    status public.oauth_client_status NOT NULL DEFAULT 'active',
    created_by_identity_id uuid
        REFERENCES public.identities (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    disabled_at timestamptz,
    deleted_at timestamptz,

    CONSTRAINT oauth_clients_id_type_uk UNIQUE (id, client_type),
    CONSTRAINT oauth_clients_client_id_chk CHECK (client_id = btrim(client_id) AND client_id <> ''),
    CONSTRAINT oauth_clients_name_chk CHECK (btrim(name) <> ''),
    CONSTRAINT oauth_clients_description_chk CHECK (
        description IS NULL OR btrim(description) <> ''
    ),
    CONSTRAINT oauth_clients_auth_method_chk CHECK (
        (client_type = 'public' AND token_endpoint_auth_method = 'none')
        OR
        (
            client_type = 'confidential'
            AND token_endpoint_auth_method = 'client_secret_basic'
        )
    ),
    CONSTRAINT oauth_clients_public_pkce_chk CHECK (
        client_type <> 'public' OR require_pkce
    ),
    CONSTRAINT oauth_clients_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT oauth_clients_state_chk CHECK (
        (status = 'active' AND disabled_at IS NULL AND deleted_at IS NULL)
        OR
        (status = 'disabled' AND disabled_at IS NOT NULL AND deleted_at IS NULL)
        OR
        (status = 'deleted' AND deleted_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS oauth_clients_client_id_uidx
    ON public.oauth_clients (client_id);

CREATE INDEX IF NOT EXISTS oauth_clients_created_by_identity_idx
    ON public.oauth_clients (created_by_identity_id)
    WHERE created_by_identity_id IS NOT NULL;

CREATE TRIGGER oauth_clients_protocol_identity_trg
    BEFORE UPDATE ON public.oauth_clients
    FOR EACH ROW
    WHEN (
        OLD.client_id IS DISTINCT FROM NEW.client_id
        OR OLD.client_type IS DISTINCT FROM NEW.client_type
        OR OLD.token_endpoint_auth_method IS DISTINCT FROM NEW.token_endpoint_auth_method
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.oauth_client_redirect_uris (
    id uuid PRIMARY KEY,
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE CASCADE,
    redirect_uri text NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT oauth_client_redirect_uris_uri_chk CHECK (
        redirect_uri = btrim(redirect_uri) AND redirect_uri <> ''
    ),
    CONSTRAINT oauth_client_redirect_uris_client_uri_uk
        UNIQUE (oauth_client_id, redirect_uri)
);

CREATE TABLE IF NOT EXISTS public.oauth_client_post_logout_redirect_uris (
    id uuid PRIMARY KEY,
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE CASCADE,
    post_logout_redirect_uri text NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT oauth_client_post_logout_redirect_uris_uri_chk CHECK (
        post_logout_redirect_uri = btrim(post_logout_redirect_uri)
        AND post_logout_redirect_uri <> ''
    ),
    CONSTRAINT oauth_client_post_logout_redirect_uris_client_uri_uk
        UNIQUE (oauth_client_id, post_logout_redirect_uri)
);

CREATE TABLE IF NOT EXISTS public.oauth_client_grant_types (
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE CASCADE,
    grant_type public.oauth_grant_type NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT oauth_client_grant_types_pk
        PRIMARY KEY (oauth_client_id, grant_type)
);

CREATE TABLE IF NOT EXISTS public.oauth_client_scopes (
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE CASCADE,
    scope text NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT oauth_client_scopes_scope_chk CHECK (
        scope = btrim(scope)
        AND scope <> ''
        AND scope !~ '[[:space:]]'
    ),
    CONSTRAINT oauth_client_scopes_pk
        PRIMARY KEY (oauth_client_id, scope)
);

CREATE TABLE IF NOT EXISTS public.oauth_client_credentials (
    id uuid PRIMARY KEY,
    oauth_client_id uuid NOT NULL,
    client_type public.oauth_client_type NOT NULL DEFAULT 'confidential',
    secret_hash text NOT NULL,
    name text,
    created_by_identity_id uuid
        REFERENCES public.identities (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz,
    revoked_at timestamptz,
    revoke_reason text,
    last_used_at timestamptz,

    CONSTRAINT oauth_client_credentials_client_fk
        FOREIGN KEY (oauth_client_id, client_type)
        REFERENCES public.oauth_clients (id, client_type)
        ON DELETE CASCADE,
    CONSTRAINT oauth_client_credentials_type_chk CHECK (
        client_type = 'confidential'
    ),
    CONSTRAINT oauth_client_credentials_hash_chk CHECK (btrim(secret_hash) <> ''),
    CONSTRAINT oauth_client_credentials_name_chk CHECK (
        name IS NULL OR btrim(name) <> ''
    ),
    CONSTRAINT oauth_client_credentials_updated_at_chk CHECK (
        updated_at >= created_at
    ),
    CONSTRAINT oauth_client_credentials_expires_at_chk CHECK (
        expires_at IS NULL OR expires_at > created_at
    ),
    CONSTRAINT oauth_client_credentials_revoke_reason_chk CHECK (
        revoked_at IS NULL OR revoke_reason IS NOT NULL
    ),
    CONSTRAINT oauth_client_credentials_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    ),
    CONSTRAINT oauth_client_credentials_last_used_at_chk CHECK (
        last_used_at IS NULL OR last_used_at >= created_at
    )
);

CREATE INDEX IF NOT EXISTS oauth_client_credentials_client_idx
    ON public.oauth_client_credentials (oauth_client_id, created_at DESC);

CREATE INDEX IF NOT EXISTS oauth_client_credentials_created_by_identity_idx
    ON public.oauth_client_credentials (created_by_identity_id)
    WHERE created_by_identity_id IS NOT NULL;

CREATE TRIGGER oauth_client_credentials_identity_trg
    BEFORE UPDATE ON public.oauth_client_credentials
    FOR EACH ROW
    WHEN (
        OLD.oauth_client_id IS DISTINCT FROM NEW.oauth_client_id
        OR OLD.client_type IS DISTINCT FROM NEW.client_type
        OR OLD.secret_hash IS DISTINCT FROM NEW.secret_hash
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.oauth_client_credentials;
DROP TABLE IF EXISTS public.oauth_client_scopes;
DROP TABLE IF EXISTS public.oauth_client_grant_types;
DROP TABLE IF EXISTS public.oauth_client_post_logout_redirect_uris;
DROP TABLE IF EXISTS public.oauth_client_redirect_uris;
DROP TABLE IF EXISTS public.oauth_clients;

DROP TYPE IF EXISTS public.oauth_client_status;
DROP TYPE IF EXISTS public.oauth_grant_type;
DROP TYPE IF EXISTS public.oauth_token_endpoint_auth_method;
DROP TYPE IF EXISTS public.oauth_client_type;

COMMIT;

