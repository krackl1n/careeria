-- +goose Up
BEGIN;

ALTER TABLE public.sessions
    DROP CONSTRAINT sessions_authenticated_at_chk,
    ADD CONSTRAINT sessions_authenticated_at_chk CHECK (authenticated_at <= created_at);

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE public.sessions
    DROP CONSTRAINT sessions_authenticated_at_chk,
    ADD CONSTRAINT sessions_authenticated_at_chk CHECK (authenticated_at >= created_at);

COMMIT;
