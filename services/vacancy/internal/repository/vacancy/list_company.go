package vacancy

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

// ListCompany returns a bounded keyset page. Public pages contain only published vacancies.
func (r *PostgresRepository) ListCompany(ctx context.Context, companyID uuid.UUID, public bool, limit int, cursor *domain.Cursor) (domain.Page, error) {
	if companyID == uuid.Nil || limit < 1 || limit > 100 {
		return domain.Page{}, fmt.Errorf("list vacancies: invalid arguments")
	}
	var at, id any
	if cursor != nil {
		if cursor.Time.IsZero() || cursor.ID == uuid.Nil {
			return domain.Page{}, fmt.Errorf("list vacancies: invalid cursor")
		}
		at, id = cursor.Time, cursor.ID
	}
	order := "v.created_at"
	filter := ""
	if public {
		order = "v.published_at"
		filter = " AND v.status = 'PUBLISHED'"
	}
	query := "SELECT " + columns + " FROM public.vacancies v WHERE v.company_id=$1" + filter +
		" AND ($2::timestamptz IS NULL OR (" + order + ",v.id)<($2::timestamptz,$3::uuid)) ORDER BY " + order + " DESC,v.id DESC LIMIT $4"
	rows, err := r.client.Executor(ctx).Query(ctx, query, companyID, at, id, limit+1)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list vacancies: %w", err)
	}
	defer rows.Close()
	var page domain.Page
	for rows.Next() {
		entity, err := scan(rows)
		if err != nil {
			return domain.Page{}, fmt.Errorf("list vacancies: %w", err)
		}
		page.Vacancies = append(page.Vacancies, entity)
	}
	if err := rows.Err(); err != nil {
		return domain.Page{}, fmt.Errorf("list vacancies: %w", err)
	}
	if len(page.Vacancies) > limit {
		page.Vacancies = page.Vacancies[:limit]
		s := page.Vacancies[limit-1].Snapshot()
		at := s.CreatedAt
		if public {
			at = s.PublishedAt
		}
		page.Next = &domain.Cursor{Time: at, ID: s.ID}
	}
	return page, nil
}
