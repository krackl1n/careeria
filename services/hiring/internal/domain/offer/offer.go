package offer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MinOfferDuration = 24 * time.Hour

type JobOffer struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Compensation  int64
	Currency      string
	Grade         string
	Status        OfferStatus
	ExpiresAt     time.Time
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func Create(
	id uuid.UUID,
	applicationID uuid.UUID,
	compensation int64,
	currency string,
	grade string,
	expiresAt time.Time,
	now time.Time,
) (*JobOffer, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if applicationID == uuid.Nil {
		return nil, ErrEmptyApplicationID
	}
	if compensation <= 0 {
		return nil, ErrInvalidCompensation
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return nil, ErrInvalidCurrency
	}
	grade = strings.TrimSpace(grade)
	if grade == "" {
		return nil, ErrEmptyGrade
	}
	if expiresAt.Before(now.Add(MinOfferDuration)) {
		return nil, ErrInvalidExpiration
	}

	return &JobOffer{
		ID:            id,
		ApplicationID: applicationID,
		Compensation:  compensation,
		Currency:      currency,
		Grade:         grade,
		Status:        StatusDraft,
		ExpiresAt:     expiresAt,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (o *JobOffer) IsExpired(now time.Time) bool {
	return now.After(o.ExpiresAt) || now.Equal(o.ExpiresAt)
}

func (o *JobOffer) Send(now time.Time) error {
	if o.Status != StatusDraft {
		return fmt.Errorf("%w: cannot send offer in status %s", ErrInvalidStatus, o.Status)
	}

	if o.IsExpired(now) {
		o.Status = StatusExpired
		o.Version++
		o.UpdatedAt = now
		return ErrOfferExpired
	}

	o.Status = StatusSent
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *JobOffer) Accept(now time.Time) error {
	if o.Status != StatusSent {
		return fmt.Errorf("%w: cannot accept offer in status %s", ErrInvalidStatus, o.Status)
	}

	if o.IsExpired(now) {
		o.Status = StatusExpired
		o.Version++
		o.UpdatedAt = now
		return ErrOfferExpired
	}

	o.Status = StatusAccepted
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *JobOffer) Decline(reason string, now time.Time) error {
	if o.Status != StatusSent {
		return fmt.Errorf("%w: cannot decline offer in status %s", ErrInvalidStatus, o.Status)
	}

	o.Status = StatusDeclined
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *JobOffer) Revoke(reason string, now time.Time) error {
	if o.Status != StatusDraft && o.Status != StatusSent {
		return fmt.Errorf("%w: cannot revoke offer in status %s", ErrInvalidStatus, o.Status)
	}

	o.Status = StatusRevoked
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *JobOffer) Expire(now time.Time) error {
	if o.Status != StatusSent {
		return fmt.Errorf("%w: cannot expire offer in status %s", ErrInvalidStatus, o.Status)
	}

	if !o.IsExpired(now) {
		return ErrOfferNotExpired
	}

	o.Status = StatusExpired
	o.Version++
	o.UpdatedAt = now
	return nil
}
