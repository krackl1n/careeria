package vacancyquery

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

type ListInput struct {
	CompanyID uuid.UUID
	View      View
	PageSize  int
	Cursor    *domain.Cursor
}

func (s *Service) ListCompany(ctx context.Context, input ListInput) (domain.Page, error) {
	if input.CompanyID == uuid.Nil || (input.View != ViewPublic && input.View != ViewCompany) ||
		input.PageSize < 1 || input.PageSize > 100 {
		return domain.Page{}, ErrInvalidInput
	}
	if input.Cursor != nil && (input.Cursor.Time.IsZero() || input.Cursor.ID == uuid.Nil) {
		return domain.Page{}, ErrInvalidInput
	}
	page, err := s.repository.ListCompany(ctx, input.CompanyID, input.View == ViewPublic, input.PageSize, input.Cursor)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list company vacancies: %w", err)
	}
	return page, nil
}
