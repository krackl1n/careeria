-- +goose Up
BEGIN;

CREATE TYPE public.authentication_level AS ENUM (
    'single_factor',
    'multi_factor'
);

CREATE TYPE public.signing_key_status AS ENUM (
    'pending',
    'active',
    'retiring',
    'retired',
    'compromised'
);

CREATE TABLE IF NOT EXISTS public.sessions (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    oauth_client_id uuid NOT NULL
        REFERENCES public.oauth_clients (id) ON DELETE RESTRICT,
    authentication_level public.authentication_level NOT NULL,
    authenticated_methods text[] NOT NULL,
    ip_address inet,
    user_agent text,
    device_name text,
    created_at timestamptz NOT NULL,
    authenticated_at timestamptz NOT NULL,
    last_seen_at timestamptz,
    idle_expires_at timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    revoke_reason text,

    CONSTRAINT sessions_methods_chk CHECK (cardinality(authenticated_methods) > 0),
    CONSTRAINT sessions_authenticated_at_chk CHECK (authenticated_at >= created_at),
    CONSTRAINT sessions_last_seen_at_chk CHECK (
        last_seen_at IS NULL OR last_seen_at >= created_at
    ),
    CONSTRAINT sessions_expiration_chk CHECK (
        idle_expires_at > created_at
        AND absolute_expires_at > created_at
        AND idle_expires_at <= absolute_expires_at
    ),
    CONSTRAINT sessions_revoke_reason_chk CHECK (
        revoked_at IS NULL OR revoke_reason IS NOT NULL
    ),
    CONSTRAINT sessions_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE INDEX IF NOT EXISTS sessions_identity_idx
    ON public.sessions (identity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS sessions_oauth_client_idx
    ON public.sessions (oauth_client_id, created_at DESC);

CREATE INDEX IF NOT EXISTS sessions_idle_expires_idx
    ON public.sessions (idle_expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS sessions_absolute_expires_idx
    ON public.sessions (absolute_expires_at)
    WHERE revoked_at IS NULL;

CREATE TRIGGER sessions_authentication_identity_trg
    BEFORE UPDATE ON public.sessions
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.oauth_client_id IS DISTINCT FROM NEW.oauth_client_id
        OR OLD.authentication_level IS DISTINCT FROM NEW.authentication_level
        OR OLD.authenticated_methods IS DISTINCT FROM NEW.authenticated_methods
        OR OLD.ip_address IS DISTINCT FROM NEW.ip_address
        OR OLD.user_agent IS DISTINCT FROM NEW.user_agent
        OR OLD.device_name IS DISTINCT FROM NEW.device_name
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
        OR OLD.authenticated_at IS DISTINCT FROM NEW.authenticated_at
        OR OLD.absolute_expires_at IS DISTINCT FROM NEW.absolute_expires_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.session_scopes (
    session_id uuid NOT NULL
        REFERENCES public.sessions (id) ON DELETE CASCADE,
    scope text NOT NULL,
    CONSTRAINT session_scopes_scope_chk CHECK (
        scope = btrim(scope)
        AND scope <> ''
        AND scope !~ '[[:space:]]'
    ),
    CONSTRAINT session_scopes_pk PRIMARY KEY (session_id, scope)
);

CREATE TABLE IF NOT EXISTS public.refresh_token_families (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL UNIQUE
        REFERENCES public.sessions (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    last_rotated_at timestamptz,
    absolute_expires_at timestamptz NOT NULL,
    reuse_detected_at timestamptz,
    revoked_at timestamptz,
    revoke_reason text,

    CONSTRAINT refresh_token_families_expiration_chk CHECK (
        absolute_expires_at > created_at
    ),
    CONSTRAINT refresh_token_families_last_rotated_at_chk CHECK (
        last_rotated_at IS NULL OR last_rotated_at >= created_at
    ),
    CONSTRAINT refresh_token_families_reuse_detected_at_chk CHECK (
        reuse_detected_at IS NULL OR reuse_detected_at >= created_at
    ),
    CONSTRAINT refresh_token_families_revoke_reason_chk CHECK (
        revoked_at IS NULL OR revoke_reason IS NOT NULL
    ),
    CONSTRAINT refresh_token_families_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE INDEX IF NOT EXISTS refresh_token_families_expires_idx
    ON public.refresh_token_families (absolute_expires_at)
    WHERE revoked_at IS NULL;

CREATE TRIGGER refresh_token_families_identity_trg
    BEFORE UPDATE ON public.refresh_token_families
    FOR EACH ROW
    WHEN (
        OLD.session_id IS DISTINCT FROM NEW.session_id
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
        OR OLD.absolute_expires_at IS DISTINCT FROM NEW.absolute_expires_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.refresh_tokens (
    id uuid PRIMARY KEY,
    family_id uuid NOT NULL
        REFERENCES public.refresh_token_families (id) ON DELETE CASCADE,
    parent_token_id uuid,
    token_hash text NOT NULL,
    issued_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    rotated_at timestamptz,
    revoked_at timestamptz,
    revoke_reason text,
    reuse_detected_at timestamptz,
    created_from_ip inet,
    created_user_agent text,

    CONSTRAINT refresh_tokens_id_family_uk UNIQUE (id, family_id),
    CONSTRAINT refresh_tokens_parent_fk
        FOREIGN KEY (parent_token_id, family_id)
        REFERENCES public.refresh_tokens (id, family_id),
    CONSTRAINT refresh_tokens_hash_chk CHECK (btrim(token_hash) <> ''),
    CONSTRAINT refresh_tokens_parent_chk CHECK (parent_token_id IS NULL OR parent_token_id <> id),
    CONSTRAINT refresh_tokens_expiration_chk CHECK (expires_at > issued_at),
    CONSTRAINT refresh_tokens_rotated_at_chk CHECK (
        rotated_at IS NULL OR rotated_at >= issued_at
    ),
    CONSTRAINT refresh_tokens_revoke_reason_chk CHECK (
        revoked_at IS NULL OR revoke_reason IS NOT NULL
    ),
    CONSTRAINT refresh_tokens_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= issued_at
    ),
    CONSTRAINT refresh_tokens_reuse_detected_at_chk CHECK (
        reuse_detected_at IS NULL OR reuse_detected_at >= issued_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_hash_uidx
    ON public.refresh_tokens (token_hash);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_current_family_uidx
    ON public.refresh_tokens (family_id)
    WHERE rotated_at IS NULL AND revoked_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_parent_uidx
    ON public.refresh_tokens (parent_token_id)
    WHERE parent_token_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS refresh_tokens_family_issued_idx
    ON public.refresh_tokens (family_id, issued_at DESC);

CREATE INDEX IF NOT EXISTS refresh_tokens_expires_idx
    ON public.refresh_tokens (expires_at)
    WHERE rotated_at IS NULL AND revoked_at IS NULL;

CREATE TRIGGER refresh_tokens_identity_trg
    BEFORE UPDATE ON public.refresh_tokens
    FOR EACH ROW
    WHEN (
        OLD.family_id IS DISTINCT FROM NEW.family_id
        OR OLD.parent_token_id IS DISTINCT FROM NEW.parent_token_id
        OR OLD.token_hash IS DISTINCT FROM NEW.token_hash
        OR OLD.issued_at IS DISTINCT FROM NEW.issued_at
        OR OLD.expires_at IS DISTINCT FROM NEW.expires_at
        OR OLD.created_from_ip IS DISTINCT FROM NEW.created_from_ip
        OR OLD.created_user_agent IS DISTINCT FROM NEW.created_user_agent
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.signing_keys (
    id uuid PRIMARY KEY,
    kid text NOT NULL,
    algorithm text NOT NULL,
    public_jwk jsonb NOT NULL,
    private_key_ref text NOT NULL,
    status public.signing_key_status NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL,
    not_before timestamptz NOT NULL,
    activated_at timestamptz,
    retiring_at timestamptz,
    retired_at timestamptz,
    expires_at timestamptz,
    compromised_at timestamptz,

    CONSTRAINT signing_keys_kid_chk CHECK (btrim(kid) <> ''),
    CONSTRAINT signing_keys_algorithm_chk CHECK (btrim(algorithm) <> ''),
    CONSTRAINT signing_keys_public_jwk_chk CHECK (
        jsonb_typeof(public_jwk) = 'object'
        AND NOT (
            public_jwk ?| ARRAY['d', 'p', 'q', 'dp', 'dq', 'qi', 'oth', 'k']
        )
    ),
    CONSTRAINT signing_keys_private_key_ref_chk CHECK (btrim(private_key_ref) <> ''),
    CONSTRAINT signing_keys_not_before_chk CHECK (not_before >= created_at),
    CONSTRAINT signing_keys_expires_at_chk CHECK (
        expires_at IS NULL OR expires_at > not_before
    ),
    CONSTRAINT signing_keys_state_timestamps_chk CHECK (
        (status <> 'active' OR activated_at IS NOT NULL)
        AND (status <> 'retiring' OR retiring_at IS NOT NULL)
        AND (status <> 'retired' OR retired_at IS NOT NULL)
        AND (status <> 'compromised' OR compromised_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS signing_keys_kid_uidx
    ON public.signing_keys (kid);

CREATE UNIQUE INDEX IF NOT EXISTS signing_keys_active_algorithm_uidx
    ON public.signing_keys (algorithm)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS signing_keys_jwks_idx
    ON public.signing_keys (status, not_before, expires_at)
    WHERE status IN ('active', 'retiring');

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.signing_keys;
DROP TABLE IF EXISTS public.refresh_tokens;
DROP TABLE IF EXISTS public.refresh_token_families;
DROP TABLE IF EXISTS public.session_scopes;
DROP TABLE IF EXISTS public.sessions;

DROP TYPE IF EXISTS public.signing_key_status;
DROP TYPE IF EXISTS public.authentication_level;

COMMIT;

