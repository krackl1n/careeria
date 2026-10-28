-- +goose Up
BEGIN;

CREATE TABLE IF NOT EXISTS public.outbox_messages (
    id uuid PRIMARY KEY,
    event_id uuid NOT NULL,
    aggregate_type varchar(255) NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NULL,
    event_type varchar(255) NOT NULL,
    payload jsonb NOT NULL,
    headers jsonb NOT NULL DEFAULT '{}'::jsonb,
    partition_key varchar(255) NULL,
    status varchar(32) NOT NULL DEFAULT 'PENDING',
    attempts integer NOT NULL DEFAULT 0,
    occurred_at timestamptz NOT NULL,
    processing_started_at timestamptz NULL,
    published_at timestamptz NULL,
    next_attempt_at timestamptz NULL,
    last_error text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT outbox_messages_aggregate_type_not_empty
        CHECK (btrim(aggregate_type) <> ''),

    CONSTRAINT outbox_messages_event_type_not_empty
        CHECK (btrim(event_type) <> ''),

    CONSTRAINT outbox_messages_payload_object
        CHECK (jsonb_typeof(payload) = 'object'),

    CONSTRAINT outbox_messages_headers_object
        CHECK (jsonb_typeof(headers) = 'object'),

    CONSTRAINT outbox_messages_version_valid
        CHECK (
            aggregate_version IS NULL
            OR aggregate_version >= 0
        ),

    CONSTRAINT outbox_messages_status_valid
        CHECK (
            status IN (
                'PENDING',
                'PROCESSING',
                'PUBLISHED',
                'FAILED',
                'DEAD'
            )
        ),

    CONSTRAINT outbox_messages_attempts_valid
        CHECK (attempts >= 0),

    CONSTRAINT outbox_messages_published_valid
        CHECK (
            status <> 'PUBLISHED'
            OR published_at IS NOT NULL
        ),

    CONSTRAINT outbox_messages_processing_valid
        CHECK (
            status <> 'PROCESSING'
            OR processing_started_at IS NOT NULL
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_outbox_messages_event_id
    ON public.outbox_messages (event_id);

CREATE INDEX IF NOT EXISTS ix_outbox_messages_aggregate
    ON public.outbox_messages (
        aggregate_type,
        aggregate_id,
        aggregate_version
    );

CREATE INDEX IF NOT EXISTS ix_outbox_messages_processing
    ON public.outbox_messages (
        status,
        next_attempt_at,
        occurred_at
    )
    WHERE status IN ('PENDING', 'FAILED');

CREATE INDEX IF NOT EXISTS ix_outbox_messages_stuck_processing
    ON public.outbox_messages (processing_started_at)
    WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS ix_outbox_messages_partition
    ON public.outbox_messages (
        partition_key,
        occurred_at
    )
    WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS ix_outbox_messages_published_at
    ON public.outbox_messages (published_at)
    WHERE status = 'PUBLISHED';

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS public.outbox_messages;

COMMIT;
