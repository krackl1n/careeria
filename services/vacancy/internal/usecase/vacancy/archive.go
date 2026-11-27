package vacancy

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (s *Service) Archive(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	return s.change(ctx, id, version, "vacancy.archived", func(entity *domain.Vacancy) error {
		return entity.Archive(version, s.clock.Now())
	})
}
