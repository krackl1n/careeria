package vacancy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (s *Service) change(ctx context.Context, id uuid.UUID, version int64, eventType string, apply func(*domain.Vacancy) error) (*domain.Vacancy, error) {
	if id == uuid.Nil || version < 1 {
		return nil, ErrInvalidInput
	}
	entity, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load vacancy: %w", err)
	}
	if err := apply(entity); err != nil {
		return nil, err
	}
	state := entity.Snapshot()
	err = s.transactions.Do(ctx, func(txCtx context.Context) error {
		if err := s.repository.Update(txCtx, entity, version); err != nil {
			return err
		}
		return s.recordEvent(txCtx, eventType, state)
	})
	if err != nil {
		return nil, fmt.Errorf("save vacancy: %w", err)
	}
	return entity, nil
}
