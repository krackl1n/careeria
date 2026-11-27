package vacancy

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
	StatusArchived  Status = "ARCHIVED"
	StatusClosed    Status = "CLOSED"
)

// State holds the aggregate data. A zero publication or closing time denotes an absent timestamp.
type State struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	AuthorID         uuid.UUID
	Details          Details
	Status           Status
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FirstPublishedAt time.Time
	PublishedAt      time.Time
	ClosedAt         time.Time
}

type Vacancy struct {
	state State
}

func NewDraft(id, companyID, authorID uuid.UUID, details Details, now time.Time) (*Vacancy, error) {
	return FromState(State{
		ID:        id,
		CompanyID: companyID,
		AuthorID:  authorID,
		Details:   details,
		Status:    StatusDraft,
		Version:   1,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	})
}

// FromState reconstitutes a persisted vacancy without advancing its version.
func FromState(state State) (*Vacancy, error) {
	if state.ID == uuid.Nil || state.CompanyID == uuid.Nil || state.AuthorID == uuid.Nil {
		return nil, fmt.Errorf("%w: vacancy, company and author IDs are required", ErrInvalidVacancy)
	}
	if state.Version <= 0 || state.CreatedAt.IsZero() || state.UpdatedAt.IsZero() || state.UpdatedAt.Before(state.CreatedAt) {
		return nil, fmt.Errorf("%w: invalid version or timestamps", ErrInvalidVacancy)
	}
	details, err := state.Details.normalized()
	if err != nil {
		return nil, err
	}
	state.Details = details
	switch state.Status {
	case StatusDraft:
		if !state.FirstPublishedAt.IsZero() || !state.PublishedAt.IsZero() {
			return nil, fmt.Errorf("%w: draft has publication timestamps", ErrInvalidVacancy)
		}
	case StatusPublished, StatusArchived, StatusClosed:
		if state.FirstPublishedAt.IsZero() || state.FirstPublishedAt.Before(state.CreatedAt) ||
			state.PublishedAt.Before(state.FirstPublishedAt) || state.PublishedAt.After(state.UpdatedAt) {
			return nil, fmt.Errorf("%w: invalid publication timestamps", ErrInvalidVacancy)
		}
		if state.Status == StatusPublished {
			if err := details.validatePublication(); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("%w: unknown status %q", ErrInvalidVacancy, state.Status)
	}
	if state.Status == StatusClosed {
		if state.ClosedAt.IsZero() || state.ClosedAt.Before(state.PublishedAt) || state.ClosedAt.After(state.UpdatedAt) {
			return nil, fmt.Errorf("%w: invalid closing time", ErrInvalidVacancy)
		}
	} else if !state.ClosedAt.IsZero() {
		return nil, fmt.Errorf("%w: open vacancy has a closing time", ErrInvalidVacancy)
	}
	return &Vacancy{state: state}, nil
}

// Snapshot returns an independent copy for persistence, reads or preparing an update.
func (v *Vacancy) Snapshot() State {
	s := v.state
	s.Details = s.Details.clone()
	return s
}

func (v *Vacancy) checkChange(expectedVersion int64, now time.Time) error {
	switch {
	case v == nil || v.state.Version <= 0:
		return ErrInvalidVacancy
	case v.state.Version != expectedVersion:
		return ErrVersionConflict
	case now.IsZero() || now.Before(v.state.UpdatedAt) || expectedVersion == math.MaxInt64:
		return fmt.Errorf("%w: invalid change time or exhausted version", ErrInvalidVacancy)
	default:
		return nil
	}
}

func (v *Vacancy) touch(now time.Time) {
	v.state.Version++
	v.state.UpdatedAt = now.UTC()
}

// UpdateDetails replaces the editable contents and skill set in a single change.
func (v *Vacancy) UpdateDetails(expectedVersion int64, details Details, now time.Time) error {
	if err := v.checkChange(expectedVersion, now); err != nil {
		return err
	}
	if v.state.Status == StatusClosed {
		return ErrInvalidTransition
	}
	next, err := details.normalized()
	if err != nil {
		return err
	}
	if v.state.Status == StatusPublished {
		if err := next.validatePublication(); err != nil {
			return err
		}
	}
	v.state.Details = next
	v.touch(now)
	return nil
}

func (v *Vacancy) Publish(expectedVersion int64, now time.Time) error {
	return v.transition(StatusDraft, StatusPublished, expectedVersion, now)
}

func (v *Vacancy) Archive(expectedVersion int64, now time.Time) error {
	return v.transition(StatusPublished, StatusArchived, expectedVersion, now)
}

func (v *Vacancy) Restore(expectedVersion int64, now time.Time) error {
	return v.transition(StatusArchived, StatusPublished, expectedVersion, now)
}

func (v *Vacancy) Close(expectedVersion int64, now time.Time) error {
	return v.transition(StatusPublished, StatusClosed, expectedVersion, now)
}

func (v *Vacancy) transition(from, to Status, expectedVersion int64, now time.Time) error {
	if err := v.checkChange(expectedVersion, now); err != nil {
		return err
	}
	if v.state.Status != from {
		return fmt.Errorf("%w: cannot change %s to %s", ErrInvalidTransition, v.state.Status, to)
	}
	switch to {
	case StatusPublished:
		if err := v.state.Details.validatePublication(); err != nil {
			return err
		}
		if from == StatusDraft {
			v.state.FirstPublishedAt = now.UTC()
		}
		v.state.PublishedAt = now.UTC()
	case StatusClosed:
		v.state.ClosedAt = now.UTC()
	}
	v.state.Status = to
	v.touch(now)
	return nil
}
