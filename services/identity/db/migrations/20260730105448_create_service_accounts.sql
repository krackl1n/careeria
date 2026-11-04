-- +goose Up
BEGIN;

CREATE TYPE public.service_account_status AS ENUM (
    'active',
    'disabled',
    'deleted'
);

CREATE TYPE public.service_account_credential_type AS ENUM (
    'shared_secret'
);

CREATE TABLE IF NOT EXISTS public.service_accounts (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    account_key text NOT NULL,
    name text NOT NULL,
    description text,
    status public.service_account_status NOT NULL DEFAULT 'active',
    created_by_identity_id uuid
        REFERENCES public.identities (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    disabled_at timestamptz,
    deleted_at timestamptz,

    CONSTRAINT service_accounts_key_chk CHECK (btrim(account_key) <> ''),
    CONSTRAINT service_accounts_name_chk CHECK (btrim(name) <> ''),
    CONSTRAINT service_accounts_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT service_accounts_state_chk CHECK (
        (status = 'active' AND disabled_at IS NULL AND deleted_at IS NULL)
        OR
        (status = 'disabled' AND disabled_at IS NOT NULL AND deleted_at IS NULL)
        OR
        (status = 'deleted' AND deleted_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS service_accounts_org_key_uidx
    ON public.service_accounts (organization_id, account_key);

CREATE INDEX IF NOT EXISTS service_accounts_org_active_idx
    ON public.service_accounts (organization_id, id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS service_accounts_deleted_at_idx
    ON public.service_accounts (deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS service_accounts_created_by_identity_idx
    ON public.service_accounts (created_by_identity_id)
    WHERE created_by_identity_id IS NOT NULL;

CREATE TRIGGER service_accounts_identity_trg
    BEFORE UPDATE ON public.service_accounts
    FOR EACH ROW
    WHEN (
        OLD.organization_id IS DISTINCT FROM NEW.organization_id
        OR OLD.account_key IS DISTINCT FROM NEW.account_key
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.service_account_credentials (
    id uuid PRIMARY KEY,
    service_account_id uuid NOT NULL
        REFERENCES public.service_accounts (id) ON DELETE CASCADE,
    credential_type public.service_account_credential_type NOT NULL DEFAULT 'shared_secret',
    secret_hash text NOT NULL,
    secret_prefix text,
    name text,
    created_by_identity_id uuid
        REFERENCES public.identities (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz,
    revoked_at timestamptz,
    revoke_reason text,
    last_used_at timestamptz,

    CONSTRAINT service_account_credentials_hash_chk CHECK (btrim(secret_hash) <> ''),
    CONSTRAINT service_account_credentials_prefix_chk CHECK (
        secret_prefix IS NULL OR btrim(secret_prefix) <> ''
    ),
    CONSTRAINT service_account_credentials_name_chk CHECK (
        name IS NULL OR btrim(name) <> ''
    ),
    CONSTRAINT service_account_credentials_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT service_account_credentials_expires_at_chk CHECK (
        expires_at IS NULL OR expires_at > created_at
    ),
    CONSTRAINT service_account_credentials_revoke_reason_chk CHECK (
        revoked_at IS NULL OR revoke_reason IS NOT NULL
    ),
    CONSTRAINT service_account_credentials_revoked_at_chk CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    ),
    CONSTRAINT service_account_credentials_last_used_at_chk CHECK (
        last_used_at IS NULL OR last_used_at >= created_at
    )
);

CREATE INDEX IF NOT EXISTS service_account_credentials_account_idx
    ON public.service_account_credentials (service_account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS service_account_credentials_created_by_identity_idx
    ON public.service_account_credentials (created_by_identity_id)
    WHERE created_by_identity_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS service_account_credentials_prefix_idx
    ON public.service_account_credentials (secret_prefix)
    WHERE secret_prefix IS NOT NULL AND revoked_at IS NULL;

CREATE TRIGGER service_account_credentials_identity_trg
    BEFORE UPDATE ON public.service_account_credentials
    FOR EACH ROW
    WHEN (
        OLD.service_account_id IS DISTINCT FROM NEW.service_account_id
        OR OLD.credential_type IS DISTINCT FROM NEW.credential_type
        OR OLD.secret_hash IS DISTINCT FROM NEW.secret_hash
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.service_account_credentials;
DROP TABLE IF EXISTS public.service_accounts;

DROP TYPE IF EXISTS public.service_account_credential_type;
DROP TYPE IF EXISTS public.service_account_status;

COMMIT;

