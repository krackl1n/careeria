-- Проект PostgreSQL-ограничений, не миграция.
ALTER TABLE notification_templates ADD CONSTRAINT ck_template_version CHECK (version > 0), ADD CONSTRAINT ck_template_key CHECK (length(btrim(template_key)) > 0);
ALTER TABLE template_revisions ADD CONSTRAINT ck_revision_number CHECK (revision > 0),
 ADD CONSTRAINT ck_revision_channel CHECK (channel IN ('IN_APP','EMAIL','PUSH','SMS')),
 ADD CONSTRAINT ck_revision_locale CHECK (length(btrim(locale)) > 0);
ALTER TABLE delivery_jobs ADD CONSTRAINT ck_delivery_version CHECK (version > 0 AND attempt_count >= 0),
 ADD CONSTRAINT ck_delivery_status CHECK (status IN ('PENDING','PROCESSING','RETRY','SUCCEEDED','FAILED','CANCELLED')),
 ADD CONSTRAINT ck_delivery_lease CHECK ((lease_token IS NULL) = (lease_until IS NULL)),
 ADD CONSTRAINT ck_delivery_processing CHECK (status <> 'PROCESSING' OR (lease_token IS NOT NULL AND lease_until IS NOT NULL));
ALTER TABLE delivery_attempts ADD CONSTRAINT ck_delivery_attempt_ordinal CHECK (ordinal > 0),
 ADD CONSTRAINT ck_delivery_attempt_time CHECK (finished_at IS NULL OR finished_at >= started_at);
ALTER TABLE delivery_jobs ADD CONSTRAINT ck_notification_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE template_revisions ADD CONSTRAINT ck_notification_template_revisions_version CHECK (version > 0);
