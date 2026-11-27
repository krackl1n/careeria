package vacancy

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (s *Service) Publish(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	return s.change(ctx, id, version, "vacancy.published", func(entity *domain.Vacancy) error {
		return entity.Publish(version, s.clock.Now())
	})
}
