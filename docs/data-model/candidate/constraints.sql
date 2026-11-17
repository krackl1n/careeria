-- Проект дополнений к DBML; не миграция. Применяется после создания таблиц.
ALTER TABLE candidate_profiles ADD CONSTRAINT ck_profile_version CHECK (version > 0),
  ADD CONSTRAINT ck_profile_name CHECK (length(btrim(display_name)) > 0),
  ADD CONSTRAINT ck_profile_archived CHECK (lifecycle <> 'ARCHIVED' OR (visibility = 'PRIVATE' AND contact_visibility = 'HIDDEN' AND search_status = 'NOT_LOOKING'));
ALTER TABLE candidate_profile_links ADD CONSTRAINT ck_profile_link_position CHECK (position >= 0);
ALTER TABLE work_experiences ADD CONSTRAINT ck_experience_dates CHECK (ended_on IS NULL OR ended_on >= started_on),
  ADD CONSTRAINT ck_experience_version CHECK (version > 0),
  ADD CONSTRAINT ck_experience_fields CHECK (length(btrim(company_name)) > 0 AND length(btrim(title)) > 0);
ALTER TABLE educations ADD CONSTRAINT ck_education_dates CHECK (ended_on IS NULL OR started_on IS NULL OR ended_on >= started_on),
  ADD CONSTRAINT ck_education_version CHECK (version > 0),
  ADD CONSTRAINT ck_education_institution CHECK (length(btrim(institution)) > 0);
ALTER TABLE candidate_skill_sets ADD CONSTRAINT ck_skill_set_version CHECK (version > 0);
ALTER TABLE candidate_skills ADD CONSTRAINT ck_skill_key CHECK (length(btrim(skill_key)) > 0 AND length(btrim(display_name)) > 0);
ALTER TABLE career_preferences ADD CONSTRAINT ck_preferences_version CHECK (version > 0),
  ADD CONSTRAINT ck_preferences_compensation CHECK (
    (compensation_min IS NULL OR compensation_min >= 0) AND (compensation_max IS NULL OR compensation_max >= 0)
    AND (compensation_min IS NULL OR compensation_max IS NULL OR compensation_min <= compensation_max)
    AND ((compensation_min IS NULL AND compensation_max IS NULL) OR (compensation_currency IS NOT NULL AND compensation_period IS NOT NULL))
  );
ALTER TABLE resumes ADD CONSTRAINT ck_resume_version CHECK (version > 0), ADD CONSTRAINT ck_resume_title CHECK (length(btrim(title)) > 0);
ALTER TABLE resume_versions ADD CONSTRAINT ck_resume_version_fields CHECK (version > 0 AND ordinal > 0),
  ADD CONSTRAINT ck_resume_ready CHECK (status <> 'READY' OR (file_id IS NOT NULL AND verified_mime_type IS NOT NULL AND verified_size_bytes IS NOT NULL AND verified_size_bytes > 0 AND verified_checksum IS NOT NULL AND ready_at IS NOT NULL));
ALTER TABLE candidate_resume_preferences ADD CONSTRAINT ck_resume_preference_version CHECK (version > 0);
ALTER TABLE resume_cleanup_jobs ADD CONSTRAINT ck_cleanup_attempts CHECK (attempts >= 0),
  ADD CONSTRAINT ck_cleanup_lease_pair CHECK ((lease_token IS NULL) = (lease_until IS NULL));
ALTER TABLE work_experiences ADD CONSTRAINT ck_candidate_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE educations ADD CONSTRAINT ck_candidate_request_pair_2 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
ALTER TABLE resumes ADD CONSTRAINT ck_candidate_request_pair_3 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
