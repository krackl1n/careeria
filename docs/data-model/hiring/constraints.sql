-- Проект ограничений, не миграция; применяется после создания таблиц DBML.
ALTER TABLE applications ADD CONSTRAINT ck_application_version CHECK (version > 0),
  ADD CONSTRAINT ck_application_stage CHECK (current_stage IN ('APPLIED','SCREENING','TECH_ASSESSMENT','INTERVIEW','OFFER_PENDING','OFFER_ACCEPTED','HIRED','REJECTED','WITHDRAWN'));
ALTER TABLE stage_transitions ADD CONSTRAINT ck_transition_ordinal CHECK (ordinal > 0);
ALTER TABLE interviews ADD CONSTRAINT ck_interview_interval CHECK (scheduled_end_at > scheduled_start_at),
  ADD CONSTRAINT ck_interview_version CHECK (version > 0),
  ADD CONSTRAINT ck_interview_status CHECK (status IN ('SCHEDULED','COMPLETED','CANCELLED'));
ALTER TABLE interview_feedback ADD CONSTRAINT ck_feedback_score CHECK (score BETWEEN 1 AND 5);
ALTER TABLE job_offers ADD CONSTRAINT ck_offer_fields CHECK (version > 0 AND compensation > 0 AND length(btrim(grade)) > 0),
  ADD CONSTRAINT ck_offer_status CHECK (status IN ('DRAFT','SENT','ACCEPTED','DECLINED','EXPIRED','REVOKED'));
ALTER TABLE applications ADD CONSTRAINT ck_hiring_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE interviews ADD CONSTRAINT ck_hiring_request_pair_2 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE job_offers ADD CONSTRAINT ck_hiring_request_pair_3 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
