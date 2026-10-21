package offer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
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
		return nil, apperr.InvalidArgument("empty_id", "offer ID must not be empty", nil)
	}
	if applicationID == uuid.Nil {
		return nil, apperr.InvalidArgument("empty_application_id", "application ID must not be empty", nil)
	}
	if compensation <= 0 {
		return nil, apperr.InvalidArgument("invalid_compensation", "compensation must be strictly positive", nil)
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return nil, apperr.InvalidArgument("invalid_currency", "currency must be a 3-letter ISO code", nil)
	}
	grade = strings.TrimSpace(grade)
	if grade == "" {
		return nil, apperr.InvalidArgument("empty_grade", "grade must not be empty", nil)
	}
	if expiresAt.Before(now.Add(MinOfferDuration)) {
		return nil, apperr.InvalidArgument(
			"invalid_expiration",
			"offer validity period must be at least 24 hours from creation",
			nil,
		)
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
		return apperr.Conflict(
			"invalid_status",
			fmt.Sprintf("cannot send offer in status %s", o.status),
			nil,
		)
	}

	if o.IsExpired(now) {
		o.status = StatusExpired
		o.version++
		o.updatedAt = now
		return apperr.Conflict("offer_expired", "cannot send an expired offer", nil)
	}

	o.status = StatusSent
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Accept(now time.Time) error {
	if o.status != StatusSent {
		return apperr.Conflict(
			"invalid_status",
			fmt.Sprintf("cannot accept offer in status %s", o.status),
			nil,
		)
	}

	if o.IsExpired(now) {
		o.status = StatusExpired
		o.version++
		o.updatedAt = now
		return apperr.Conflict("offer_expired", "cannot accept an expired offer", nil)
	}

	o.status = StatusAccepted
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Decline(reason string, now time.Time) error {
	if o.status != StatusSent {
		return apperr.Conflict(
			"invalid_status",
			fmt.Sprintf("cannot decline offer in status %s", o.status),
			nil,
		)
	}

	o.status = StatusDeclined
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Revoke(reason string, now time.Time) error {
	if o.status != StatusDraft && o.status != StatusSent {
		return apperr.Conflict(
			"invalid_status",
			fmt.Sprintf("cannot revoke offer in status %s", o.status),
			nil,
		)
	}

	o.status = StatusRevoked
	o.version++
	o.updatedAt = now
	return nil
}

func (o *JobOffer) Expire(now time.Time) error {
	if o.status != StatusSent {
		return apperr.Conflict(
			"invalid_status",
			fmt.Sprintf("cannot expire offer in status %s", o.status),
			nil,
		)
	}

	if !o.IsExpired(now) {
		return apperr.Conflict("not_expired", "offer expiration time has not been reached yet", nil)
	}

	o.status = StatusExpired
	o.version++
	o.updatedAt = now
	return nil
}

