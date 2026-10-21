package offer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MinOfferDuration = 24 * time.Hour

type JobOffer struct {
	id            uuid.UUID
	applicationID uuid.UUID
	compensation  int64
	currency      string
	grade         string
	status        OfferStatus
	expiresAt     time.Time
	version       int64
	createdAt     time.Time
	updatedAt     time.Time
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
		id:            id,
		applicationID: applicationID,
		compensation:  compensation,
		currency:      currency,
		grade:         grade,
		status:        StatusDraft,
		expiresAt:     expiresAt,
		version:       1,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	applicationID uuid.UUID,
	compensation int64,
	currency string,
	grade string,
	status OfferStatus,
	expiresAt time.Time,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) *JobOffer {
	return &JobOffer{
		id:            id,
		applicationID: applicationID,
		compensation:  compensation,
		currency:      currency,
		grade:         grade,
		status:        status,
		expiresAt:     expiresAt,
		version:       version,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

func (o *JobOffer) ID() uuid.UUID {
	return o.id
}

func (o *JobOffer) ApplicationID() uuid.UUID {
	return o.applicationID
}

func (o *JobOffer) Compensation() int64 {
	return o.compensation
}

func (o *JobOffer) Currency() string {
	return o.currency
}

func (o *JobOffer) Grade() string {
	return o.grade
}

func (o *JobOffer) Status() OfferStatus {
	return o.status
}

func (o *JobOffer) ExpiresAt() time.Time {
	return o.expiresAt
}

func (o *JobOffer) Version() int64 {
	return o.version
}

func (o *JobOffer) CreatedAt() time.Time {
	return o.createdAt
}

func (o *JobOffer) UpdatedAt() time.Time {
	return o.updatedAt
}

func (o *JobOffer) IsExpired(now time.Time) bool {
	return now.After(o.expiresAt) || now.Equal(o.expiresAt)
}

func (o *JobOffer) Send(now time.Time) error {
	if o.status != StatusDraft {
		return fmt.Errorf("%w: cannot send offer in status %s", ErrInvalidStatus, o.status)
	}

	if o.IsExpired(now) {
		o.status = StatusExpired
		o.version++
		o.updatedAt = now
		return ErrOfferExpired
	}

	o.status = StatusSent
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Accept(now time.Time) error {
	if o.status != StatusSent {
		return fmt.Errorf("%w: cannot accept offer in status %s", ErrInvalidStatus, o.status)
	}

	if o.IsExpired(now) {
		o.status = StatusExpired
		o.version++
		o.updatedAt = now
		return ErrOfferExpired
	}

	o.status = StatusAccepted
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Decline(reason string, now time.Time) error {
	if o.status != StatusSent {
		return fmt.Errorf("%w: cannot decline offer in status %s", ErrInvalidStatus, o.status)
	}

	o.status = StatusDeclined
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Revoke(reason string, now time.Time) error {
	if o.status != StatusDraft && o.status != StatusSent {
		return fmt.Errorf("%w: cannot revoke offer in status %s", ErrInvalidStatus, o.status)
	}

	o.status = StatusRevoked
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Expire(now time.Time) error {
	if o.status != StatusSent {
		return fmt.Errorf("%w: cannot expire offer in status %s", ErrInvalidStatus, o.status)
	}

	if !o.IsExpired(now) {
		return ErrOfferNotExpired
	}

	o.status = StatusExpired
	o.version++
	o.updatedAt = now
	return nil
}
