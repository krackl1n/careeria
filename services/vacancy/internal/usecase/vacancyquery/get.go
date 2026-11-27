package vacancyquery

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (s *Service) Get(ctx context.Context, id uuid.UUID, view View) (*domain.Vacancy, error) {
	if id == uuid.Nil || (view != ViewPublic && view != ViewCompany) {
		return nil, ErrInvalidInput
	}
	var entity *domain.Vacancy
	var err error
	if view == ViewCompany {
		entity, err = s.repository.Get(ctx, id)
	} else {
		entity, err = s.repository.GetPublished(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get vacancy: %w", err)
	}
	return entity, nil
}
