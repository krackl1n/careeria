-- +goose Up
CREATE TYPE public.vacancy_status AS ENUM ('DRAFT', 'PUBLISHED', 'ARCHIVED', 'CLOSED');
CREATE TYPE public.work_format AS ENUM ('OFFICE', 'REMOTE', 'HYBRID');

CREATE TABLE IF NOT EXISTS public.vacancies (
    id uuid PRIMARY KEY,
    company_id uuid NOT NULL,
    author_id uuid NOT NULL,
    create_request_id uuid UNIQUE,
    create_request_hash text,
    title text NOT NULL,
    description text NOT NULL,
    status public.vacancy_status NOT NULL DEFAULT 'DRAFT',
    required_experience text,
    employment_type text,
    work_format public.work_format NOT NULL,
    location text,
    compensation_min numeric(18,2),
    compensation_max numeric(18,2),
    compensation_currency char(3),
    compensation_period text,
    compensation_tax_basis text,
    compensation_bonus text,
    version bigint NOT NULL DEFAULT 1,
    first_published_at timestamptz,
    published_at timestamptz,
    closed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_vacancy_version CHECK (version > 0),
    CONSTRAINT ck_vacancy_title CHECK (btrim(title) <> ''),
    CONSTRAINT ck_vacancy_request_pair CHECK ((create_request_id IS NULL) = (create_request_hash IS NULL)),
    CONSTRAINT ck_vacancy_compensation CHECK (
        (compensation_min IS NULL OR compensation_min >= 0)
        AND (compensation_max IS NULL OR compensation_max >= 0)
        AND (compensation_min IS NULL OR compensation_max IS NULL OR compensation_min <= compensation_max)
        AND ((compensation_min IS NULL AND compensation_max IS NULL)
             OR (compensation_currency IS NOT NULL AND compensation_period IS NOT NULL))
        AND (compensation_tax_basis IS NULL OR compensation_tax_basis IN ('GROSS', 'NET'))
    ),
    CONSTRAINT ck_vacancy_publication_time CHECK (
        status = 'DRAFT'
        OR (first_published_at IS NOT NULL AND published_at IS NOT NULL AND published_at >= first_published_at)
    ),
    CONSTRAINT ck_vacancy_closed_time CHECK (status <> 'CLOSED' OR closed_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS ix_vacancies_company_list ON public.vacancies (company_id, status, created_at, id);
CREATE INDEX IF NOT EXISTS ix_vacancies_public_list ON public.vacancies (status, published_at, id);

CREATE TABLE IF NOT EXISTS public.vacancy_skills (
    vacancy_id uuid NOT NULL REFERENCES public.vacancies (id),
    skill_key text NOT NULL,
    display_name text NOT NULL,
    PRIMARY KEY (vacancy_id, skill_key),
    CONSTRAINT ck_vacancy_skill CHECK (btrim(skill_key) <> '' AND btrim(display_name) <> '')
);

-- +goose Down
DROP TABLE IF EXISTS public.vacancy_skills;
DROP TABLE IF EXISTS public.vacancies;
DROP TYPE IF EXISTS public.work_format;
DROP TYPE IF EXISTS public.vacancy_status;
