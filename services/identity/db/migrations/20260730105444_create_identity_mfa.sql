-- +goose Up
BEGIN;

CREATE TYPE public.mfa_factor_type AS ENUM (
    'totp',
    'webauthn'
);

CREATE TYPE public.totp_algorithm AS ENUM (
    'sha1',
    'sha256',
    'sha512'
);

CREATE TABLE IF NOT EXISTS public.mfa_factors (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    factor_type public.mfa_factor_type NOT NULL,
    name text,
    is_primary boolean NOT NULL DEFAULT false,
    confirmed_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_used_at timestamptz,
    disabled_at timestamptz,

    CONSTRAINT mfa_factors_name_chk CHECK (name IS NULL OR btrim(name) <> ''),
    CONSTRAINT mfa_factors_primary_chk CHECK (
        NOT is_primary OR (confirmed_at IS NOT NULL AND disabled_at IS NULL)
    ),
    CONSTRAINT mfa_factors_updated_at_chk CHECK (updated_at >= created_at),
    CONSTRAINT mfa_factors_confirmed_at_chk CHECK (
        confirmed_at IS NULL OR confirmed_at >= created_at
    ),
    CONSTRAINT mfa_factors_last_used_at_chk CHECK (
        last_used_at IS NULL OR last_used_at >= created_at
    ),
    CONSTRAINT mfa_factors_disabled_at_chk CHECK (
        disabled_at IS NULL OR disabled_at >= created_at
    ),
    CONSTRAINT mfa_factors_id_type_uk UNIQUE (id, factor_type)
);

CREATE UNIQUE INDEX IF NOT EXISTS mfa_factors_primary_uidx
    ON public.mfa_factors (identity_id)
    WHERE is_primary AND confirmed_at IS NOT NULL AND disabled_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mfa_factors_enabled_totp_uidx
    ON public.mfa_factors (identity_id)
    WHERE factor_type = 'totp' AND disabled_at IS NULL;

CREATE INDEX IF NOT EXISTS mfa_factors_identity_idx
    ON public.mfa_factors (identity_id);

CREATE TRIGGER mfa_factors_identity_type_trg
    BEFORE UPDATE ON public.mfa_factors
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.factor_type IS DISTINCT FROM NEW.factor_type
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.totp_factors (
    factor_id uuid PRIMARY KEY,
    factor_type public.mfa_factor_type NOT NULL DEFAULT 'totp',
    secret_encrypted bytea NOT NULL,
    encryption_key_ref text NOT NULL,
    algorithm public.totp_algorithm NOT NULL DEFAULT 'sha1',
    digits smallint NOT NULL DEFAULT 6,
    period_seconds smallint NOT NULL DEFAULT 30,
    last_used_time_step bigint,
    updated_at timestamptz NOT NULL,

    CONSTRAINT totp_factors_type_chk CHECK (factor_type = 'totp'),
    CONSTRAINT totp_factors_factor_fk FOREIGN KEY (factor_id, factor_type)
        REFERENCES public.mfa_factors (id, factor_type) ON DELETE CASCADE,
    CONSTRAINT totp_factors_secret_chk CHECK (octet_length(secret_encrypted) > 0),
    CONSTRAINT totp_factors_key_ref_chk CHECK (btrim(encryption_key_ref) <> ''),
    CONSTRAINT totp_factors_digits_chk CHECK (digits IN (6, 8)),
    CONSTRAINT totp_factors_period_chk CHECK (period_seconds > 0),
    CONSTRAINT totp_factors_time_step_chk CHECK (
        last_used_time_step IS NULL OR last_used_time_step >= 0
    )
);

CREATE TABLE IF NOT EXISTS public.webauthn_credentials (
    factor_id uuid PRIMARY KEY,
    factor_type public.mfa_factor_type NOT NULL DEFAULT 'webauthn',
    credential_id bytea NOT NULL,
    public_key bytea NOT NULL,
    sign_count bigint NOT NULL DEFAULT 0,
    aaguid uuid,
    transports text[] NOT NULL DEFAULT '{}',
    backup_eligible boolean NOT NULL DEFAULT false,
    backup_state boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL,
    last_used_at timestamptz,

    CONSTRAINT webauthn_credentials_type_chk CHECK (factor_type = 'webauthn'),
    CONSTRAINT webauthn_credentials_factor_fk FOREIGN KEY (factor_id, factor_type)
        REFERENCES public.mfa_factors (id, factor_type) ON DELETE CASCADE,
    CONSTRAINT webauthn_credentials_credential_id_chk CHECK (octet_length(credential_id) > 0),
    CONSTRAINT webauthn_credentials_public_key_chk CHECK (octet_length(public_key) > 0),
    CONSTRAINT webauthn_credentials_sign_count_chk CHECK (sign_count >= 0),
    CONSTRAINT webauthn_credentials_last_used_at_chk CHECK (
        last_used_at IS NULL OR last_used_at >= created_at
    ),
    CONSTRAINT webauthn_credentials_backup_state_chk CHECK (
        NOT backup_state OR backup_eligible
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS webauthn_credentials_credential_id_uidx
    ON public.webauthn_credentials (credential_id);

CREATE TABLE IF NOT EXISTS public.recovery_code_sets (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL
        REFERENCES public.identities (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    expires_at timestamptz,
    disabled_at timestamptz,

    CONSTRAINT recovery_code_sets_expires_at_chk CHECK (
        expires_at IS NULL OR expires_at > created_at
    ),
    CONSTRAINT recovery_code_sets_disabled_at_chk CHECK (
        disabled_at IS NULL OR disabled_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS recovery_code_sets_active_uidx
    ON public.recovery_code_sets (identity_id)
    WHERE disabled_at IS NULL;

CREATE INDEX IF NOT EXISTS recovery_code_sets_identity_idx
    ON public.recovery_code_sets (identity_id);

CREATE TRIGGER recovery_code_sets_identity_trg
    BEFORE UPDATE ON public.recovery_code_sets
    FOR EACH ROW
    WHEN (
        OLD.identity_id IS DISTINCT FROM NEW.identity_id
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

CREATE TABLE IF NOT EXISTS public.recovery_codes (
    id uuid PRIMARY KEY,
    set_id uuid NOT NULL
        REFERENCES public.recovery_code_sets (id) ON DELETE CASCADE,
    code_hash text NOT NULL,
    created_at timestamptz NOT NULL,
    used_at timestamptz,

    CONSTRAINT recovery_codes_hash_chk CHECK (btrim(code_hash) <> ''),
    CONSTRAINT recovery_codes_used_at_chk CHECK (
        used_at IS NULL OR used_at >= created_at
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS recovery_codes_set_hash_uidx
    ON public.recovery_codes (set_id, code_hash);

CREATE INDEX IF NOT EXISTS recovery_codes_unused_idx
    ON public.recovery_codes (set_id)
    WHERE used_at IS NULL;

CREATE TRIGGER recovery_codes_identity_trg
    BEFORE UPDATE ON public.recovery_codes
    FOR EACH ROW
    WHEN (
        OLD.set_id IS DISTINCT FROM NEW.set_id
        OR OLD.code_hash IS DISTINCT FROM NEW.code_hash
        OR OLD.created_at IS DISTINCT FROM NEW.created_at
    )
    EXECUTE FUNCTION public.reject_immutable_update();

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.recovery_codes;
DROP TABLE IF EXISTS public.recovery_code_sets;
DROP TABLE IF EXISTS public.webauthn_credentials;
DROP TABLE IF EXISTS public.totp_factors;
DROP TABLE IF EXISTS public.mfa_factors;

DROP TYPE IF EXISTS public.totp_algorithm;
DROP TYPE IF EXISTS public.mfa_factor_type;

COMMIT;

