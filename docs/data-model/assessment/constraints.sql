-- Проект ограничений, не миграция; применяется после таблиц из DBML.
ALTER TABLE assessment_templates ADD CONSTRAINT ck_template_version CHECK (version > 0), ADD CONSTRAINT ck_template_title CHECK (length(btrim(title)) > 0);
ALTER TABLE template_revisions ADD CONSTRAINT ck_revision_limits CHECK (ordinal > 0 AND time_limit_minutes BETWEEN 10 AND 4320 AND passing_threshold BETWEEN 0 AND 100);
ALTER TABLE assessment_tasks ADD CONSTRAINT ck_task_values CHECK (max_score > 0 AND position >= 0);
ALTER TABLE assessment_sessions ADD CONSTRAINT ck_session_version CHECK (version > 0),
  ADD CONSTRAINT ck_session_status CHECK (status IN ('SCHEDULED','IN_PROGRESS','SUBMITTED','EVALUATED','EXPIRED')),
  ADD CONSTRAINT ck_session_time CHECK ((expires_at IS NULL OR (started_at IS NOT NULL AND expires_at > started_at)) AND (submitted_at IS NULL OR started_at IS NOT NULL)),
  ADD CONSTRAINT ck_session_result CHECK (final_score IS NULL OR final_score BETWEEN 0 AND 100),
  ADD CONSTRAINT ck_session_evaluated CHECK (status <> 'EVALUATED' OR (final_score IS NOT NULL AND is_passed IS NOT NULL));
ALTER TABLE task_submissions ADD CONSTRAINT ck_submission_version CHECK (version > 0),
  ADD CONSTRAINT ck_submission_status CHECK (status IN ('PENDING','RUNNING','PASSED','FAILED','COMPILATION_ERROR','TIMEOUT')),
  ADD CONSTRAINT ck_submission_metrics CHECK ((execution_time_ms IS NULL OR execution_time_ms >= 0) AND (peak_memory_bytes IS NULL OR peak_memory_bytes >= 0));
ALTER TABLE runner_jobs ADD CONSTRAINT ck_runner_job_attempts CHECK (attempts >= 0), ADD CONSTRAINT ck_runner_job_lease CHECK ((lease_token IS NULL) = (lease_until IS NULL));
ALTER TABLE assessment_manual_reviews ADD CONSTRAINT ck_review_version CHECK (version > 0);
ALTER TABLE review_scores ADD CONSTRAINT ck_review_score CHECK (score BETWEEN 0 AND 100 AND length(btrim(criterion_key)) > 0);
ALTER TABLE assessment_templates ADD CONSTRAINT ck_assessment_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE assessment_sessions ADD CONSTRAINT ck_assessment_request_pair_2 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE task_submissions ADD CONSTRAINT ck_assessment_request_pair_3 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE template_revisions ADD CONSTRAINT ck_assessment_template_revisions_version CHECK (version > 0);
