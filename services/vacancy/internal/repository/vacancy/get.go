package vacancy

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Vacancy, error) {
	entity, err := scan(r.client.Executor(ctx).QueryRow(ctx, "SELECT "+columns+" FROM public.vacancies v WHERE v.id=$1", id))
	if err != nil {
		return nil, fmt.Errorf("get vacancy: %w", err)
	}
	return entity, nil
}
