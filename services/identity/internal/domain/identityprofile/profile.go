package identityprofile

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type Profile struct {
	IdentityID identitydomain.ID

	FamilyName string
	GivenName  string
	MiddleName *string

	Source         Source
	AssuranceLevel AssuranceLevel

	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(
	identityID identitydomain.ID,
	familyName string,
	givenName string,
	middleName *string,
	now time.Time,
) (*Profile, error) {
	if !identityID.IsValid() {
		return nil, errors.New("create identity profile: identity identifier is required")
	}
	familyName, givenName, middleName, err := normalizeNames(familyName, givenName, middleName)
	if err != nil {
		return nil, fmt.Errorf("create identity profile: %w", err)
	}
	if now.IsZero() {
		return nil, errors.New("create identity profile: time is invalid")
	}

	return &Profile{
		IdentityID: identityID,

		FamilyName: familyName,
		GivenName:  givenName,
		MiddleName: domain.ClonePointer(middleName),

		Source:         SourceSelfDeclared,
		AssuranceLevel: AssuranceLevelSelfDeclared,

		Version:   InitialVersion,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Profile) ApplyVerifiedNames(
	familyName string,
	givenName string,
	middleName *string,
	now time.Time,
) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("apply verified identity profile names: %w", err)
	}
	if p.Source == SourceProvisioned {
		return fmt.Errorf("apply verified identity profile names: %w", ErrExternallyManaged)
	}
	familyName, givenName, middleName, err := normalizeNames(familyName, givenName, middleName)
	if err != nil {
		return fmt.Errorf("apply verified identity profile names: %w", err)
	}
	if p.FamilyName == familyName &&
		p.GivenName == givenName &&
		domain.EqualOptionalText(p.MiddleName, middleName) &&
		p.Source == SourceVerified &&
		p.AssuranceLevel == AssuranceLevelIdentityVerified {
		return nil
	}

	p.FamilyName = familyName
	p.GivenName = givenName
	p.MiddleName = domain.ClonePointer(middleName)
	p.Source = SourceVerified
	p.AssuranceLevel = AssuranceLevelIdentityVerified
	p.Version++
	p.UpdatedAt = now

	return nil
}

func (p *Profile) ensureChangeAllowed(now time.Time) error {
	if p == nil {
		return errors.New("validate identity profile for change: profile is nil")
	}
	if now.IsZero() || !now.After(p.UpdatedAt) {
		return errors.New("validate identity profile for change: time is invalid")
	}

	return nil
}

func normalizeNames(familyName, givenName string, middleName *string) (string, string, *string, error) {
	familyName, ok := domain.NormalizeRequiredText(familyName)
	if !ok {
		return "", "", nil, errors.New("family name is required")
	}
	givenName, ok = domain.NormalizeRequiredText(givenName)
	if !ok {
		return "", "", nil, errors.New("given name is required")
	}

	return familyName, givenName, domain.NormalizeOptionalText(middleName), nil
}
