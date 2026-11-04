package identityprofile

import (
	"time"

	"github.com/google/uuid"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

const (
	AggregateType    eventdomain.AggregateType = "identity_profile"
	EventTypeUpdated eventdomain.Type          = "IdentityProfileUpdated"
)

type UpdatedEvent struct {
	IdentitySub    string         `json:"identity_sub"`
	FamilyName     string         `json:"family_name"`
	GivenName      string         `json:"given_name"`
	MiddleName     *string        `json:"middle_name,omitempty"`
	Source         Source         `json:"source"`
	AssuranceLevel AssuranceLevel `json:"assurance_level"`
	Version        int64          `json:"version"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (p *Profile) UpdatedEvent() UpdatedEvent {
	return UpdatedEvent{
		IdentitySub:    p.IdentityID.UUID().String(),
		FamilyName:     p.FamilyName,
		GivenName:      p.GivenName,
		MiddleName:     p.MiddleName,
		Source:         p.Source,
		AssuranceLevel: p.AssuranceLevel,
		Version:        p.Version,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func AggregateID(identityID uuid.UUID) uuid.UUID { return identityID }
