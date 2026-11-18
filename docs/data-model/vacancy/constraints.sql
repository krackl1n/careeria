-- Проект дополнений к DBML; не миграция. Применяется после создания таблиц.
ALTER TABLE vacancies ADD CONSTRAINT ck_vacancy_version CHECK (version > 0),
  ADD CONSTRAINT ck_vacancy_title CHECK (length(btrim(title)) > 0),
  ADD CONSTRAINT ck_vacancy_compensation CHECK (
    (compensation_min IS NULL OR compensation_min >= 0) AND (compensation_max IS NULL OR compensation_max >= 0)
    AND (compensation_min IS NULL OR compensation_max IS NULL OR compensation_min <= compensation_max)
    AND ((compensation_min IS NULL AND compensation_max IS NULL) OR (compensation_currency IS NOT NULL AND compensation_period IS NOT NULL))
    AND (compensation_tax_basis IS NULL OR compensation_tax_basis IN ('GROSS', 'NET'))
  ),
  ADD CONSTRAINT ck_vacancy_publication_time CHECK (status = 'DRAFT' OR (first_published_at IS NOT NULL AND published_at IS NOT NULL AND published_at >= first_published_at)),
  ADD CONSTRAINT ck_vacancy_closed_time CHECK (status <> 'CLOSED' OR closed_at IS NOT NULL);
ALTER TABLE vacancy_skills ADD CONSTRAINT ck_vacancy_skill CHECK (length(btrim(skill_key)) > 0 AND length(btrim(display_name)) > 0);
ALTER TABLE vacancies ADD CONSTRAINT ck_vacancy_request_pair_1 CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL));
