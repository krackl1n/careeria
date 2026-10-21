-- +goose Up
BEGIN;

CREATE TABLE IF NOT EXISTS public.email_verification_challenges (
    id uuid PRIMARY KEY,
    email_id uuid NOT NULL
        REFERENCES public.identity_emails (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL,
    expires_at timestamptz NOT NULL,
    last_attempt_at timestamptz,
    consumed_at timestamptz,
    revoked_at timestamptz,
    requested_from_ip inet,
    consumed_from_ip inet,
    created_at timestamptz NOT NULL,

    CONSTRAINT email_verification_challenges_hash_chk CHECK (btrim(token_hash) <> ''),
    CONSTRAINT email_verification_challenges_attempts_chk CHECK (
        attempts >= 0 AND max_attempts > 0 AND attempts <= max_attempts
    ),
    CONSTRAINT email_verification_challenges_last_attempt_at_chk CHECK (
        (attempts = 0 AND last_attempt_at IS NULL)
        OR
        (
            attempts > 0
            AND last_attempt_at IS NOT NULL
            AND last_attempt_at >= created_at
        )
    ),
    CONSTRAINT email_verification_challenges_expires_at_chk CHECK (expires_at > created_at),
    CONSTRAINT email_verification_challenges_state_chk CHECK (
        NOT (consumed_at IS NOT NULL AND revoked_at IS NOT NULL)
    ),
    CONSTRAINT email_verification_challenges_consumed_at_chk CHECK (
        consumed_at IS NULL OR consumed_at >= created_at
    ),
    CONSTRAINT email_verification_challenges_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS email_verification_challenges_token_uidx
    ON public.email_verification_challenges (token_hash);

CREATE INDEX IF NOT EXISTS email_verification_challenges_email_idx
    ON public.email_verification_challenges (email_id, created_at DESC);

CREATE INDEX IF NOT EXISTS email_verification_challenges_expires_idx
    ON public.email_verification_challenges (expires_at)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS public.phone_verification_challenges (
    id uuid PRIMARY KEY,
    phone_id uuid NOT NULL
        REFERENCES public.identity_phones (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL,
    expires_at timestamptz NOT NULL,
    last_attempt_at timestamptz,
    consumed_at timestamptz,
    revoked_at timestamptz,
    requested_from_ip inet,
    consumed_from_ip inet,
    created_at timestamptz NOT NULL,

    CONSTRAINT phone_verification_challenges_hash_chk CHECK (btrim(token_hash) <> ''),
    CONSTRAINT phone_verification_challenges_attempts_chk CHECK (
        attempts >= 0 AND max_attempts > 0 AND attempts <= max_attempts
    ),
    CONSTRAINT phone_verification_challenges_last_attempt_at_chk CHECK (
        (attempts = 0 AND last_attempt_at IS NULL)
        OR
        (
            attempts > 0
            AND last_attempt_at IS NOT NULL
            AND last_attempt_at >= created_at
        )
    ),
    CONSTRAINT phone_verification_challenges_expires_at_chk CHECK (expires_at > created_at),
    CONSTRAINT phone_verification_challenges_state_chk CHECK (
        NOT (consumed_at IS NOT NULL AND revoked_at IS NOT NULL)
    ),
    CONSTRAINT phone_verification_challenges_consumed_at_chk CHECK (
        consumed_at IS NULL OR consumed_at >= created_at
    ),
    CONSTRAINT phone_verification_challenges_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS phone_verification_challenges_token_uidx
    ON public.phone_verification_challenges (token_hash);

CREATE INDEX IF NOT EXISTS phone_verification_challenges_phone_idx
    ON public.phone_verification_challenges (phone_id, created_at DESC);

CREATE INDEX IF NOT EXISTS phone_verification_challenges_expires_idx
    ON public.phone_verification_challenges (expires_at)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS public.password_reset_challenges (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL,
    expires_at timestamptz NOT NULL,
    last_attempt_at timestamptz,
    consumed_at timestamptz,
    revoked_at timestamptz,
    requested_from_ip inet,
    consumed_from_ip inet,
    created_at timestamptz NOT NULL,

    CONSTRAINT password_reset_challenges_hash_chk CHECK (btrim(token_hash) <> ''),
    CONSTRAINT password_reset_challenges_attempts_chk CHECK (
        attempts >= 0 AND max_attempts > 0 AND attempts <= max_attempts
    ),
    CONSTRAINT password_reset_challenges_last_attempt_at_chk CHECK (
        (attempts = 0 AND last_attempt_at IS NULL)
        OR
        (
            attempts > 0
            AND last_attempt_at IS NOT NULL
            AND last_attempt_at >= created_at
        )
    ),
    CONSTRAINT password_reset_challenges_expires_at_chk CHECK (expires_at > created_at),
    CONSTRAINT password_reset_challenges_state_chk CHECK (
        NOT (consumed_at IS NOT NULL AND revoked_at IS NOT NULL)
    ),
    CONSTRAINT password_reset_challenges_consumed_at_chk CHECK (
        consumed_at IS NULL OR consumed_at >= created_at
    ),
    CONSTRAINT password_reset_challenges_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_challenges_token_uidx
    ON public.password_reset_challenges (token_hash);

CREATE INDEX IF NOT EXISTS password_reset_challenges_identity_idx
    ON public.password_reset_challenges (identity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS password_reset_challenges_expires_idx
    ON public.password_reset_challenges (expires_at)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.password_reset_challenges;
DROP TABLE IF EXISTS public.phone_verification_challenges;
DROP TABLE IF EXISTS public.email_verification_challenges;

COMMIT;

