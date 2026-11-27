package vacancy

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (r *PostgresRepository) GetByCreateRequest(ctx context.Context, id uuid.UUID) (*domain.Vacancy, string, error) {
	var vacancyID uuid.UUID
	var hash string
	err := r.client.Executor(ctx).QueryRow(ctx, "SELECT id, create_request_hash FROM public.vacancies WHERE create_request_id=$1", id).Scan(&vacancyID, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("get create request: %w", err)
	}
	entity, err := r.Get(ctx, vacancyID)
	return entity, hash, err
}
