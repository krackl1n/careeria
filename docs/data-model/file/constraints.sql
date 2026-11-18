-- Проект PostgreSQL-ограничений, не миграция.
ALTER TABLE files ADD CONSTRAINT ck_file_version CHECK (version > 0),
 ADD CONSTRAINT ck_file_state CHECK (state IN ('UPLOADING','CHECKING','READY','REJECTED','DELETING','DELETED')),
 ADD CONSTRAINT ck_file_ready CHECK (state <> 'READY' OR (verified_mime_type IS NOT NULL AND verified_size_bytes IS NOT NULL AND verified_size_bytes > 0 AND verified_checksum IS NOT NULL AND storage_version_id IS NOT NULL AND ready_at IS NOT NULL)),
 ADD CONSTRAINT ck_file_deleted CHECK (state <> 'DELETED' OR deleted_at IS NOT NULL);
ALTER TABLE upload_sessions ADD CONSTRAINT ck_upload_version CHECK (version > 0),
 ADD CONSTRAINT ck_upload_expiry CHECK (expires_at > created_at),
 ADD CONSTRAINT ck_upload_state CHECK (state IN ('INITIALIZING','UPLOADING','COMPLETING','COMPLETED','CANCELLED','EXPIRED'));
ALTER TABLE file_checks ADD CONSTRAINT ck_file_check_state CHECK (status IN ('PENDING','RUNNING','PASSED','FAILED','ERROR')),
 ADD CONSTRAINT ck_file_check_time CHECK (completed_at IS NULL OR (started_at IS NOT NULL AND completed_at >= started_at));
ALTER TABLE file_bindings ADD CONSTRAINT ck_file_binding_status CHECK (status IN ('PENDING','ACTIVE','RELEASED')),
 ADD CONSTRAINT ck_file_binding_released CHECK (status <> 'RELEASED' OR (released_at IS NOT NULL AND release_operation_id IS NOT NULL));
ALTER TABLE file_cleanup_jobs ADD CONSTRAINT ck_file_cleanup_attempts CHECK (attempts >= 0 AND version > 0),
 ADD CONSTRAINT ck_file_cleanup_lease CHECK ((lease_token IS NULL) = (lease_until IS NULL));
ALTER TABLE file_bindings ADD CONSTRAINT ck_file_file_bindings_version CHECK (version > 0);
