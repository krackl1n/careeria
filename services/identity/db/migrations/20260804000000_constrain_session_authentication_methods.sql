-- +goose Up
BEGIN;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'public.sessions'::regclass
          AND conname = 'sessions_known_authentication_methods_chk'
    ) THEN
        ALTER TABLE public.sessions
            ADD CONSTRAINT sessions_known_authentication_methods_chk
            CHECK (
                authenticated_methods <@ ARRAY[
                    'password',
                    'totp',
                    'webauthn',
                    'recovery_code'
                ]::text[]
            );
    END IF;
END;
$$;
-- +goose StatementEnd

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE public.sessions
    DROP CONSTRAINT IF EXISTS sessions_known_authentication_methods_chk;

COMMIT;
