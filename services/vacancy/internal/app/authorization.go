package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/ctxmeta"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	commands "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	queries "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
)

type commandService interface {
	CreateDraft(context.Context, commands.CreateDraftInput) (*domain.Vacancy, error)
	UpdateDetails(context.Context, uuid.UUID, int64, commands.DetailsPatch) (*domain.Vacancy, error)
	Publish(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Archive(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Restore(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
	Close(context.Context, uuid.UUID, int64) (*domain.Vacancy, error)
}

type queryService interface {
	Get(context.Context, uuid.UUID, queries.View) (*domain.Vacancy, error)
	ListCompany(context.Context, queries.ListInput) (domain.Page, error)
}

type authorizedVacancyService struct {
	commands   commandService
	queries    queryService
	authorizer domain.Authorizer
}

func (s *authorizedVacancyService) require(ctx context.Context, companyID uuid.UUID, kind string, resourceID uuid.UUID, permission string) error {
	principal, ok := ctxmeta.PrincipalFromContext(ctx)
	if !ok || principal.SubjectID == "" {
		return domain.ErrUnauthenticated
	}
	userID, err := uuid.Parse(principal.SubjectID)
	if err != nil || userID == uuid.Nil {
		return domain.ErrUnauthenticated
	}
	err = s.authorizer.Require(ctx, domain.Access{
		UserID:     userID,
		CompanyID:  companyID,
		ResourceID: resourceID,
		Resource:   kind,
		Permission: permission,
	})
	switch {
	case errors.Is(err, domain.ErrPermissionDenied):
		return domain.ErrPermissionDenied
	case err != nil:
		return fmt.Errorf("%w: %w", domain.ErrAuthorizationUnavailable, err)
	default:
		return nil
	}
}

func (s *authorizedVacancyService) requireVacancy(ctx context.Context, id uuid.UUID, permission string) (*domain.Vacancy, error) {
	entity, err := s.queries.Get(ctx, id, queries.ViewCompany)
	if err != nil {
		return nil, fmt.Errorf("load vacancy for authorization: %w", err)
	}
	if err := s.require(ctx, entity.Snapshot().CompanyID, "vacancy", id, permission); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *authorizedVacancyService) CreateDraft(ctx context.Context, input commands.CreateDraftInput) (*domain.Vacancy, error) {
	if err := s.require(ctx, input.CompanyID, "company", input.CompanyID, "can_create_vacancy"); err != nil {
		return nil, err
	}
	return s.commands.CreateDraft(ctx, input)
}

func (s *authorizedVacancyService) UpdateDetails(ctx context.Context, id uuid.UUID, version int64, patch commands.DetailsPatch) (*domain.Vacancy, error) {
	if _, err := s.requireVacancy(ctx, id, "can_edit"); err != nil {
		return nil, err
	}
	return s.commands.UpdateDetails(ctx, id, version, patch)
}

func (s *authorizedVacancyService) Publish(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	if _, err := s.requireVacancy(ctx, id, "can_publish"); err != nil {
		return nil, err
	}
	return s.commands.Publish(ctx, id, version)
}

func (s *authorizedVacancyService) Archive(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	if _, err := s.requireVacancy(ctx, id, "can_archive"); err != nil {
		return nil, err
	}
	return s.commands.Archive(ctx, id, version)
}

func (s *authorizedVacancyService) Restore(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	if _, err := s.requireVacancy(ctx, id, "can_restore"); err != nil {
		return nil, err
	}
	return s.commands.Restore(ctx, id, version)
}

func (s *authorizedVacancyService) Close(ctx context.Context, id uuid.UUID, version int64) (*domain.Vacancy, error) {
	if _, err := s.requireVacancy(ctx, id, "can_close"); err != nil {
		return nil, err
	}
	return s.commands.Close(ctx, id, version)
}

func (s *authorizedVacancyService) Get(ctx context.Context, id uuid.UUID, view queries.View) (*domain.Vacancy, error) {
	if view == queries.ViewCompany {
		return s.requireVacancy(ctx, id, "can_view")
	}
	return s.queries.Get(ctx, id, view)
}

func (s *authorizedVacancyService) ListCompany(ctx context.Context, input queries.ListInput) (domain.Page, error) {
	if input.View == queries.ViewCompany {
		if err := s.require(ctx, input.CompanyID, "company", input.CompanyID, "can_view_vacancies"); err != nil {
			return domain.Page{}, err
		}
	}
	return s.queries.ListCompany(ctx, input)
}
