package vacancy

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type WorkFormat string

const (
	WorkOffice WorkFormat = "OFFICE"
	WorkRemote WorkFormat = "REMOTE"
	WorkHybrid WorkFormat = "HYBRID"
)

type SkillRequirement struct {
	Key         string
	DisplayName string
}

type Details struct {
	Title              string
	Description        string
	WorkFormat         WorkFormat
	Location           *string
	RequiredExperience *string
	EmploymentType     *string
	Compensation       *Compensation
	Skills             []SkillRequirement
}

func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func (d Details) clone() Details {
	d.Location = clonePtr(d.Location)
	d.RequiredExperience = clonePtr(d.RequiredExperience)
	d.EmploymentType = clonePtr(d.EmploymentType)
	d.Compensation = d.Compensation.clone()
	d.Skills = slices.Clone(d.Skills)
	return d
}

func (d Details) normalized() (Details, error) {
	d = d.clone()
	d.Title = strings.TrimSpace(d.Title)
	if d.Title == "" {
		return Details{}, fmt.Errorf("%w: title is required", ErrInvalidDetails)
	}
	switch d.WorkFormat {
	case WorkOffice, WorkRemote, WorkHybrid:
	default:
		return Details{}, fmt.Errorf("%w: unknown work format %q", ErrInvalidDetails, d.WorkFormat)
	}
	if err := d.Compensation.validate(); err != nil {
		return Details{}, err
	}

	keys := make(map[string]struct{}, len(d.Skills))
	for i := range d.Skills {
		skill := &d.Skills[i]
		skill.Key = strings.ToLower(strings.TrimSpace(skill.Key))
		skill.DisplayName = strings.TrimSpace(skill.DisplayName)
		if skill.Key == "" || skill.DisplayName == "" {
			return Details{}, fmt.Errorf("%w: skill key and display name are required", ErrInvalidDetails)
		}
		if _, exists := keys[skill.Key]; exists {
			return Details{}, fmt.Errorf("%w: duplicate skill %q", ErrInvalidDetails, skill.Key)
		}
		keys[skill.Key] = struct{}{}
	}
	return d, nil
}

func (d Details) validatePublication() error {
	switch {
	case utf8.RuneCountInString(d.Title) < 5:
		return fmt.Errorf("%w: title must contain at least 5 characters", ErrNotPublishable)
	case utf8.RuneCountInString(strings.TrimSpace(d.Description)) < 50:
		return fmt.Errorf("%w: description must contain at least 50 characters", ErrNotPublishable)
	case len(d.Skills) == 0:
		return fmt.Errorf("%w: at least one skill is required", ErrNotPublishable)
	default:
		return nil
	}
}
