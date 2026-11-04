-- +goose Up
BEGIN;

CREATE TYPE public.security_event_result AS ENUM (
    'success',
    'failure',
    'blocked'
);

CREATE TYPE public.security_actor_type AS ENUM (
    'anonymous',
    'identity',
    'service_account',
    'administrator',
    'system'
);

CREATE TABLE IF NOT EXISTS public.security_events (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    result public.security_event_result NOT NULL,
    actor_type public.security_actor_type NOT NULL,
    identity_id uuid
        REFERENCES public.identities (id) ON DELETE SET NULL,
    service_account_id uuid
        REFERENCES public.service_accounts (id) ON DELETE SET NULL,
    session_id uuid
        REFERENCES public.sessions (id) ON DELETE SET NULL,
    oauth_client_id uuid
        REFERENCES public.oauth_clients (id) ON DELETE SET NULL,
    identifier_hash bytea,
    ip_address inet,
    user_agent text,
    correlation_id uuid,
    occurred_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}',

    CONSTRAINT security_events_event_type_chk CHECK (btrim(event_type) <> ''),
    CONSTRAINT security_events_actor_chk CHECK (
        (
            actor_type = 'anonymous'
            AND identity_id IS NULL
            AND service_account_id IS NULL
        )
        OR
        (
            actor_type IN ('identity', 'administrator')
            AND identity_id IS NOT NULL
            AND service_account_id IS NULL
        )
        OR
        (
            actor_type = 'service_account'
            AND identity_id IS NULL
            AND service_account_id IS NOT NULL
        )
        OR
        (
            actor_type = 'system'
            AND identity_id IS NULL
            AND service_account_id IS NULL
        )
    ),
    CONSTRAINT security_events_metadata_chk CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS security_events_identity_idx
    ON public.security_events (identity_id, occurred_at DESC)
    WHERE identity_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS security_events_service_account_idx
    ON public.security_events (service_account_id, occurred_at DESC)
    WHERE service_account_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS security_events_session_idx
    ON public.security_events (session_id, occurred_at DESC)
    WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS security_events_oauth_client_idx
    ON public.security_events (oauth_client_id, occurred_at DESC)
    WHERE oauth_client_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS security_events_type_idx
    ON public.security_events (event_type, occurred_at DESC);

CREATE INDEX IF NOT EXISTS security_events_identifier_idx
    ON public.security_events (identifier_hash, occurred_at DESC)
    WHERE identifier_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS security_events_occurred_at_idx
    ON public.security_events (occurred_at);

CREATE INDEX IF NOT EXISTS security_events_correlation_idx
    ON public.security_events (correlation_id, occurred_at)
    WHERE correlation_id IS NOT NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.security_events;

DROP TYPE IF EXISTS public.security_actor_type;
DROP TYPE IF EXISTS public.security_event_result;

COMMIT;

