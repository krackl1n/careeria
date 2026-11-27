package vacancy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	"time"
)

func scan(row pgx.Row) (*domain.Vacancy, error) {
	var s domain.State
	var first, published, closed *time.Time
	var min, max *int64
	var currency, period, tax, bonus *string
	var skills []byte
	err := row.Scan(&s.ID, &s.CompanyID, &s.AuthorID, &s.Details.Title, &s.Details.Description,
		&s.Details.WorkFormat, &s.Details.Location, &s.Details.RequiredExperience, &s.Details.EmploymentType,
		&s.Status, &s.Version, &s.CreatedAt, &s.UpdatedAt, &first, &published, &closed,
		&min, &max, &currency, &period, &tax, &bonus, &skills)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if first != nil {
		s.FirstPublishedAt = *first
	}
	if published != nil {
		s.PublishedAt = *published
	}
	if closed != nil {
		s.ClosedAt = *closed
	}
	if min != nil || max != nil || currency != nil || period != nil || tax != nil || bonus != nil {
		c := &domain.Compensation{Bonus: bonus}
		if min != nil {
			value := domain.Amount(*min)
			c.Minimum = &value
		}
		if max != nil {
			value := domain.Amount(*max)
			c.Maximum = &value
		}
		if currency != nil {
			c.Currency = *currency
		}
		if period != nil {
			c.Period = *period
		}
		if tax != nil {
			c.TaxBasis = domain.TaxBasis(*tax)
		}
		s.Details.Compensation = c
	}
	if err := json.Unmarshal(skills, &s.Details.Skills); err != nil {
		return nil, fmt.Errorf("decode skills: %w", err)
	}
	return domain.FromState(s)
}

func values(s domain.State) ([]any, error) {
	skillSet := s.Details.Skills
	if skillSet == nil {
		skillSet = []domain.SkillRequirement{}
	}
	skills, err := json.Marshal(skillSet)
	if err != nil {
		return nil, err
	}
	var min, max any
	var currency, period, tax, bonus any
	if c := s.Details.Compensation; c != nil {
		if c.Minimum != nil {
			min = int64(*c.Minimum)
		}
		if c.Maximum != nil {
			max = int64(*c.Maximum)
		}
		if c.Currency != "" {
			currency = c.Currency
		}
		if c.Period != "" {
			period = c.Period
		}
		if c.TaxBasis != "" {
			tax = string(c.TaxBasis)
		}
		bonus = c.Bonus
	}
	var first, published, closed any
	if !s.FirstPublishedAt.IsZero() {
		first = s.FirstPublishedAt
	}
	if !s.PublishedAt.IsZero() {
		published = s.PublishedAt
	}
	if !s.ClosedAt.IsZero() {
		closed = s.ClosedAt
	}
	return []any{s.ID, s.CompanyID, s.AuthorID, s.Details.Title, s.Details.Description,
		string(s.Details.WorkFormat), s.Details.Location, s.Details.RequiredExperience, s.Details.EmploymentType,
		string(s.Status), s.Version, s.CreatedAt, s.UpdatedAt, first, published, closed,
		min, max, currency, period, tax, bonus, skills}, nil
}
