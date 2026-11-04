-- +goose Up
BEGIN;

-- Current-state cut-over snapshot for Documents projection rebuilds.
INSERT INTO public.outbox_messages (
    id, event_id, aggregate_type, aggregate_id, aggregate_version,
    event_type, payload, headers, partition_key, status, occurred_at,
    created_at, updated_at
)
SELECT
    gen_random_uuid(), gen_random_uuid(), 'identity_profile', profile.identity_id, profile.version,
    'IdentityProfileUpdated',
    jsonb_build_object(
        'identity_sub', profile.identity_id::text,
        'family_name', profile.family_name,
        'given_name', profile.given_name,
        'middle_name', profile.middle_name,
        'source', profile.source,
        'assurance_level', profile.assurance_level,
        'version', profile.version,
        'created_at', profile.created_at,
        'updated_at', profile.updated_at
    ),
    jsonb_build_object('projection_snapshot', '20260909100000'),
    profile.identity_id::text, 'PENDING', now(), now(), now()
FROM public.identity_profiles AS profile;

COMMIT;

-- +goose Down
-- Published snapshot events cannot be retracted from downstream consumers.
SELECT 1;
