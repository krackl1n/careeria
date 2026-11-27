package vacancyquery

import (
	"context"
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

var ErrInvalidInput = errors.New("invalid query input")

type Repository interface {
	Get(context.Context, uuid.UUID) (*domain.Vacancy, error)
	GetPublished(context.Context, uuid.UUID) (*domain.Vacancy, error)
	ListCompany(context.Context, uuid.UUID, bool, int, *domain.Cursor) (domain.Page, error)
}

type Service struct{ repository Repository }

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("vacancy query repository is required")
	}
	return &Service{repository: repo}, nil
}

type View int

const (
	ViewPublic View = iota
	ViewCompany
)
