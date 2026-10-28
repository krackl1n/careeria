package identityprofile

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestNewProfileUsesSelfDeclaredTrust(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	middleName := " Петрович "
	profile, err := New(
		identitydomain.ID(uuid.New()),
		" Иванов ",
		" Иван ",
		&middleName,
		now,
	)
	require.NoError(t, err)
	require.Equal(t, "Иванов", profile.FamilyName)
	require.Equal(t, "Иван", profile.GivenName)
	require.Equal(t, "Петрович", *profile.MiddleName)
	require.Equal(t, SourceSelfDeclared, profile.Source)
	require.Equal(t, AssuranceLevelSelfDeclared, profile.AssuranceLevel)
	require.Equal(t, InitialVersion, profile.Version)
	require.Equal(t, now, profile.CreatedAt)
	require.Equal(t, now, profile.UpdatedAt)
}

func TestNewProfileRejectsInvalidArguments(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	validName := "Иван"

	_, err := New(identitydomain.ID{}, validName, validName, nil, now)
	require.ErrorContains(t, err, "identity identifier is required")
	_, err = New(identitydomain.ID(uuid.New()), " ", validName, nil, now)
	require.ErrorContains(t, err, "family name is required")
	_, err = New(identitydomain.ID(uuid.New()), validName, validName, nil, time.Time{})
	require.ErrorContains(t, err, "time is invalid")
}

func TestApplyVerifiedNamesChangesTrustAndVersion(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	profile, err := New(
		identitydomain.ID(uuid.New()),
		"Иванов",
		"Иван",
		nil,
		now,
	)
	require.NoError(t, err)
	middleName := " Петрович "

	require.NoError(t, profile.ApplyVerifiedNames(
		" Петров ",
		" Пётр ",
		&middleName,
		now.Add(time.Minute),
	))
	require.Equal(t, "Петров", profile.FamilyName)
	require.Equal(t, "Пётр", profile.GivenName)
	require.Equal(t, "Петрович", *profile.MiddleName)
	require.Equal(t, SourceVerified, profile.Source)
	require.Equal(t, AssuranceLevelIdentityVerified, profile.AssuranceLevel)
	require.Equal(t, int64(2), profile.Version)
	require.Equal(t, now.Add(time.Minute), profile.UpdatedAt)
}

func TestApplyVerifiedNamesRejectsProvisionedProfile(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	profile := &Profile{
		IdentityID:     identitydomain.ID(uuid.New()),
		FamilyName:     "Иванов",
		GivenName:      "Иван",
		Source:         SourceProvisioned,
		AssuranceLevel: AssuranceLevelOrganizationAsserted,
		Version:        3,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := profile.ApplyVerifiedNames(
		"Петров",
		"Пётр",
		nil,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrExternallyManaged)
	require.Equal(t, "Иванов", profile.FamilyName)
	require.Equal(t, int64(3), profile.Version)
}

func TestApplyVerifiedNamesRequiresIncreasingTime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	profile, err := New(
		identitydomain.ID(uuid.New()),
		"Иванов",
		"Иван",
		nil,
		now,
	)
	require.NoError(t, err)

	err = profile.ApplyVerifiedNames(
		"Петров",
		"Пётр",
		nil,
		now,
	)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrExternallyManaged))
	require.Equal(t, InitialVersion, profile.Version)
}

func TestApplyIdenticalVerifiedNamesIsNoop(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	profile := &Profile{
		IdentityID:     identitydomain.ID(uuid.New()),
		FamilyName:     "Иванов",
		GivenName:      "Иван",
		Source:         SourceVerified,
		AssuranceLevel: AssuranceLevelIdentityVerified,
		Version:        2,
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Minute),
	}

	require.NoError(t, profile.ApplyVerifiedNames(
		"Иванов",
		"Иван",
		nil,
		now.Add(2*time.Minute),
	))
	require.Equal(t, int64(2), profile.Version)
	require.Equal(t, now.Add(time.Minute), profile.UpdatedAt)
}

func TestEnumValueObjects(t *testing.T) {
	t.Parallel()

	for _, source := range []Source{SourceSelfDeclared, SourceProvisioned, SourceVerified} {
		parsed, err := ParseSource(string(source))
		require.NoError(t, err)
		require.Equal(t, source, parsed)
	}
	_, err := ParseSource("unknown")
	require.Error(t, err)

	for _, level := range []AssuranceLevel{
		AssuranceLevelSelfDeclared,
		AssuranceLevelOrganizationAsserted,
		AssuranceLevelIdentityVerified,
	} {
		parsed, err := ParseAssuranceLevel(string(level))
		require.NoError(t, err)
		require.Equal(t, level, parsed)
	}
	_, err = ParseAssuranceLevel("unknown")
	require.Error(t, err)
}
