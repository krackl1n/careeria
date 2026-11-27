package vacancy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

// Field distinguishes an omitted change from an explicit nil for nullable fields.
type Field[T any] struct {
	Set   bool
	Value T
}

type DetailsPatch struct {
	Title              Field[string]
	Description        Field[string]
	WorkFormat         Field[domain.WorkFormat]
	Location           Field[*string]
	RequiredExperience Field[*string]
	EmploymentType     Field[*string]
	Compensation       Field[*domain.Compensation]
	Skills             Field[[]domain.SkillRequirement]
}

func (s *Service) UpdateDetails(ctx context.Context, id uuid.UUID, version int64, patch DetailsPatch) (*domain.Vacancy, error) {
	if !patch.Title.Set && !patch.Description.Set && !patch.WorkFormat.Set && !patch.Location.Set &&
		!patch.RequiredExperience.Set && !patch.EmploymentType.Set && !patch.Compensation.Set && !patch.Skills.Set {
		return nil, fmt.Errorf("%w: empty update", ErrInvalidInput)
	}
	return s.change(ctx, id, version, "vacancy.updated", func(entity *domain.Vacancy) error {
		details := entity.Snapshot().Details
		if patch.Title.Set {
			details.Title = patch.Title.Value
		}
		if patch.Description.Set {
			details.Description = patch.Description.Value
		}
		if patch.WorkFormat.Set {
			details.WorkFormat = patch.WorkFormat.Value
		}
		if patch.Location.Set {
			details.Location = patch.Location.Value
		}
		if patch.RequiredExperience.Set {
			details.RequiredExperience = patch.RequiredExperience.Value
		}
		if patch.EmploymentType.Set {
			details.EmploymentType = patch.EmploymentType.Value
		}
		if patch.Compensation.Set {
			details.Compensation = patch.Compensation.Value
		}
		if patch.Skills.Set {
			details.Skills = patch.Skills.Value
		}
		return entity.UpdateDetails(version, details, s.clock.Now())
	})
}
