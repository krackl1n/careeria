package vacancy

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/krackl1n/careeria/postgres"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (r *PostgresRepository) Create(ctx context.Context, entity *domain.Vacancy, requestID uuid.UUID, requestHash string) error {
	if entity == nil || requestID == uuid.Nil || requestHash == "" {
		return fmt.Errorf("create vacancy: invalid arguments")
	}
	s := entity.Snapshot()
	if s.Status != domain.StatusDraft || s.Version != 1 {
		return fmt.Errorf("create vacancy: draft version 1 required")
	}
	args, err := values(s)
	if err != nil {
		return err
	}
	args = append(args, requestID, requestHash)
	_, err = r.client.Executor(ctx).Exec(ctx, `
WITH inserted AS (
 INSERT INTO public.vacancies (id,company_id,author_id,title,description,work_format,location,required_experience,employment_type,status,version,created_at,updated_at,first_published_at,published_at,closed_at,compensation_min,compensation_max,compensation_currency,compensation_period,compensation_tax_basis,compensation_bonus,create_request_id,create_request_hash)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::numeric/100,$18::numeric/100,$19,$20,$21,$22,$24,$25)
 RETURNING id
)
INSERT INTO public.vacancy_skills (vacancy_id,skill_key,display_name)
SELECT inserted.id, skill."Key", skill."DisplayName" FROM inserted
CROSS JOIN jsonb_to_recordset(COALESCE($23::jsonb,'[]'::jsonb)) AS skill("Key" text,"DisplayName" text)`, args...)
	if postgres.IsUniqueViolation(err) {
		return fmt.Errorf("create vacancy: %w", domain.ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("create vacancy: %w", err)
	}
	return nil
}
