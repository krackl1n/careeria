package identityprofile

import (
	"errors"
)

const InitialVersion int64 = 1

type Source string

const (
	SourceSelfDeclared Source = "self_declared"
	SourceProvisioned  Source = "provisioned"
	SourceVerified     Source = "verified"
)

func ParseSource(value string) (Source, error) {
	source := Source(value)
	switch source {
	case SourceSelfDeclared, SourceProvisioned, SourceVerified:
		return source, nil
	default:
		return "", errors.New("unknown identity profile source")
	}
}

type AssuranceLevel string

const (
	AssuranceLevelSelfDeclared         AssuranceLevel = "self_declared"
	AssuranceLevelOrganizationAsserted AssuranceLevel = "organization_asserted"
	AssuranceLevelIdentityVerified     AssuranceLevel = "identity_verified"
)

func ParseAssuranceLevel(value string) (AssuranceLevel, error) {
	level := AssuranceLevel(value)
	switch level {
	case AssuranceLevelSelfDeclared,
		AssuranceLevelOrganizationAsserted,
		AssuranceLevelIdentityVerified:
		return level, nil
	default:
		return "", errors.New("unknown identity assurance level")
	}
}
