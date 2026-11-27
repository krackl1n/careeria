package vacancy

import (
	"context"
	"fmt"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (r *PostgresRepository) Update(ctx context.Context, entity *domain.Vacancy, expectedVersion int64) error {
	if entity == nil || expectedVersion < 1 {
		return fmt.Errorf("update vacancy: invalid arguments")
	}
	s := entity.Snapshot()
	if s.Version-1 != expectedVersion {
		return domain.ErrVersionConflict
	}
	args, err := values(s)
	if err != nil {
		return err
	}
	args = append(args, expectedVersion)
	var updated bool
	err = r.client.Executor(ctx).QueryRow(ctx, `
WITH updated AS (
 UPDATE public.vacancies SET title=$4,description=$5,work_format=$6,location=$7,required_experience=$8,
 employment_type=$9,status=$10,version=$11,updated_at=$13,first_published_at=$14,published_at=$15,closed_at=$16,
 compensation_min=$17::numeric/100,compensation_max=$18::numeric/100,compensation_currency=$19,
 compensation_period=$20,compensation_tax_basis=$21,compensation_bonus=$22
 WHERE id=$1 AND company_id=$2 AND author_id=$3 AND created_at=$12 AND version=$24
 RETURNING id
), skills AS (
 SELECT skill."Key" AS key, skill."DisplayName" AS name FROM
 jsonb_to_recordset(COALESCE($23::jsonb,'[]'::jsonb)) AS skill("Key" text,"DisplayName" text)
), removed AS (
 DELETE FROM public.vacancy_skills s USING updated u WHERE s.vacancy_id=u.id
 AND NOT EXISTS (SELECT 1 FROM skills WHERE key=s.skill_key)
), saved AS (
 INSERT INTO public.vacancy_skills (vacancy_id,skill_key,display_name)
 SELECT u.id, skills.key, skills.name FROM updated u CROSS JOIN skills
 ON CONFLICT (vacancy_id,skill_key) DO UPDATE SET display_name=EXCLUDED.display_name
)
SELECT EXISTS(SELECT 1 FROM updated)`, args...).Scan(&updated)
	if err != nil {
		return fmt.Errorf("update vacancy: %w", err)
	}
	if !updated {
		return domain.ErrVersionConflict
	}
	return nil
}
