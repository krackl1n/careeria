-- +goose Up
BEGIN;

CREATE TYPE public.identity_status AS ENUM (
    'pending',
    'active',
    'disabled',
    'deleted'
);

CREATE TABLE IF NOT EXISTS public.identities (
    id uuid PRIMARY KEY,
    status public.identity_status NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,

    CONSTRAINT identities_deleted_state_chk CHECK (
        (status = 'deleted' AND deleted_at IS NOT NULL)
        OR
        (status <> 'deleted' AND deleted_at IS NULL)
    ),
    CONSTRAINT identities_updated_at_chk CHECK (updated_at >= created_at)
);

CREATE INDEX IF NOT EXISTS identities_status_idx
    ON public.identities (status);

CREATE INDEX IF NOT EXISTS identities_deleted_at_idx
    ON public.identities (deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.identity_emails (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    email text NOT NULL,
    is_primary boolean NOT NULL DEFAULT false,
    verified_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,

    CONSTRAINT identity_emails_normalized_chk CHECK (
        email = lower(btrim(email))
        AND position('@' IN email) > 1
    ),
    CONSTRAINT identity_emails_primary_chk CHECK (
        NOT is_primary OR deleted_at IS NULL
    ),
    CONSTRAINT identity_emails_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT identity_emails_verified_at_chk CHECK (
        verified_at IS NULL OR verified_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS identity_emails_email_uidx
    ON public.identity_emails (email);

CREATE UNIQUE INDEX IF NOT EXISTS identity_emails_primary_uidx
    ON public.identity_emails (identity_id)
    WHERE is_primary AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS identity_emails_identity_idx
    ON public.identity_emails (identity_id);

CREATE TABLE IF NOT EXISTS public.identity_phones (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    phone text NOT NULL,
    is_primary boolean NOT NULL DEFAULT false,
    verified_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,

    CONSTRAINT identity_phones_e164_chk CHECK (
        phone ~ '^[+][1-9][0-9]{7,14}$'
    ),
    CONSTRAINT identity_phones_primary_chk CHECK (
        NOT is_primary OR deleted_at IS NULL
    ),
    CONSTRAINT identity_phones_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT identity_phones_verified_at_chk CHECK (
        verified_at IS NULL OR verified_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS identity_phones_phone_uidx
    ON public.identity_phones (phone);

CREATE UNIQUE INDEX IF NOT EXISTS identity_phones_primary_uidx
    ON public.identity_phones (identity_id)
    WHERE is_primary AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS identity_phones_identity_idx
    ON public.identity_phones (identity_id);

-- +goose StatementBegin
CREATE FUNCTION public.reject_immutable_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'immutable columns of % cannot be changed', TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER identity_emails_reassignment_trg
    BEFORE UPDATE ON public.identity_emails
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.email IS DISTINCT FROM NEW.email
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TRIGGER identity_phones_reassignment_trg
    BEFORE UPDATE ON public.identity_phones
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.phone IS DISTINCT FROM NEW.phone
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.external_identity_accounts (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    provider text NOT NULL,
    subject text NOT NULL,
    linked_at timestamptz NOT NULL,
    last_authenticated_at timestamptz,
    disabled_at timestamptz,
    deleted_at timestamptz,

    CONSTRAINT external_identity_accounts_provider_chk CHECK (btrim(provider) <> ''),
    CONSTRAINT external_identity_accounts_subject_chk CHECK (btrim(subject) <> ''),
    CONSTRAINT external_identity_accounts_disabled_at_chk CHECK (
        disabled_at IS NULL OR disabled_at >= linked_at
    ),
    CONSTRAINT external_identity_accounts_deleted_at_chk CHECK (
        deleted_at IS NULL
        OR (
            deleted_at >= linked_at
            AND (disabled_at IS NULL OR deleted_at >= disabled_at)
        )
    ),
    CONSTRAINT external_identity_accounts_last_auth_chk CHECK (
        last_authenticated_at IS NULL OR last_authenticated_at >= linked_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS external_identity_accounts_provider_subject_uidx
    ON public.external_identity_accounts (provider, subject);

CREATE INDEX IF NOT EXISTS external_identity_accounts_identity_idx
    ON public.external_identity_accounts (identity_id);

CREATE TRIGGER external_identity_accounts_identity_trg
    BEFORE UPDATE ON public.external_identity_accounts
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.provider IS DISTINCT FROM NEW.provider
        OR OLD.subject IS DISTINCT FROM NEW.subject
        OR OLD.linked_at IS DISTINCT FROM NEW.linked_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.identity_password_credentials (
    identity_id uuid PRIMARY KEY
        REFERENCES public.identities (id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    must_change boolean NOT NULL DEFAULT false,
    password_updated_at timestamptz NOT NULL,
    expires_at timestamptz,
    compromised_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT identity_password_credentials_hash_chk CHECK (btrim(password_hash) <> ''),
    CONSTRAINT identity_password_credentials_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT identity_password_credentials_password_updated_at_chk CHECK (
        password_updated_at >= created_at
    ),
    CONSTRAINT identity_password_credentials_expires_at_chk CHECK (
        expires_at IS NULL OR expires_at > password_updated_at
    ),
    CONSTRAINT identity_password_credentials_compromised_at_chk CHECK (
        compromised_at IS NULL OR compromised_at >= created_at
    )
);

CREATE TABLE IF NOT EXISTS public.identity_password_auth_state (
    identity_id uuid PRIMARY KEY
        REFERENCES public.identities (id) ON DELETE CASCADE,
    failed_attempts integer NOT NULL DEFAULT 0,
    last_failed_at timestamptz,
    locked_until timestamptz,
    lock_reason text,
    updated_at timestamptz NOT NULL,

    CONSTRAINT identity_password_auth_state_attempts_chk CHECK (failed_attempts >= 0),
    CONSTRAINT identity_password_auth_state_lock_reason_chk CHECK (
        (locked_until IS NULL) = (lock_reason IS NULL)
    ),
    CONSTRAINT identity_password_auth_state_last_failed_at_chk CHECK (
        last_failed_at IS NULL OR last_failed_at <= updated_at
    )
);

CREATE INDEX IF NOT EXISTS identity_password_auth_state_locked_until_idx
    ON public.identity_password_auth_state (locked_until)
    WHERE locked_until IS NOT NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.identity_password_auth_state;
DROP TABLE IF EXISTS public.identity_password_credentials;
DROP TABLE IF EXISTS public.external_identity_accounts;
DROP TABLE IF EXISTS public.identity_phones;
DROP TABLE IF EXISTS public.identity_emails;
DROP TABLE IF EXISTS public.identities;

DROP FUNCTION IF EXISTS public.reject_immutable_update();

DROP TYPE IF EXISTS public.identity_status;

COMMIT;
