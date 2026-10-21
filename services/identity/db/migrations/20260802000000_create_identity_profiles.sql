-- +goose Up
BEGIN;

CREATE TYPE public.identity_profile_source AS ENUM (
    'self_declared',
    'provisioned',
    'verified'
);

CREATE TYPE public.identity_assurance_level AS ENUM (
    'self_declared',
    'organization_asserted',
    'identity_verified'
);

CREATE TABLE IF NOT EXISTS public.identity_profiles (
    identity_id uuid PRIMARY KEY
        REFERENCES public.identities (id) ON DELETE CASCADE,

    family_name text NOT NULL,
    given_name text NOT NULL,
    middle_name text,

    source public.identity_profile_source NOT NULL DEFAULT 'self_declared',
    assurance_level public.identity_assurance_level NOT NULL DEFAULT 'self_declared',

    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT identity_profiles_family_name_chk CHECK (
        family_name = btrim(family_name)
        AND family_name <> ''
    ),
    CONSTRAINT identity_profiles_given_name_chk CHECK (
        given_name = btrim(given_name)
        AND given_name <> ''
    ),
    CONSTRAINT identity_profiles_middle_name_chk CHECK (
        middle_name IS NULL
        OR (
            middle_name = btrim(middle_name)
            AND middle_name <> ''
        )
    ),
    CONSTRAINT identity_profiles_version_chk CHECK (version >= 1),
    CONSTRAINT identity_profiles_updated_at_chk CHECK (updated_at >= created_at)
);

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.identity_profiles;
DROP TYPE IF EXISTS public.identity_assurance_level;
DROP TYPE IF EXISTS public.identity_profile_source;

COMMIT;
