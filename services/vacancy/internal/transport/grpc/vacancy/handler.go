package vacancy

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	usecase "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	query "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
)

type Service interface {
	CreateDraft(context.Context, usecase.CreateDraftInput) (*domain.Vacancy, error)
	UpdateDetails(context.Context, uuid.UUID, int64, usecase.DetailsPatch) (*domain.Vacancy, error)
	Publish(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Archive(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Restore(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Close(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)

	Get(context.Context, uuid.UUID, query.View) (*domain.Vacancy, error)
	ListCompany(context.Context, query.ListInput) (domain.Page, error)
}

type Handler struct {
	pb.UnimplementedVacancyServiceServer
	pb.UnimplementedVacancyQueryServiceServer
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}
