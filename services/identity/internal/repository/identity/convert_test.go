package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestToDomain(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	verifiedAt := now.Add(time.Minute)
	lastAuthenticatedAt := now.Add(2 * time.Minute)
	disabledAt := now.Add(3 * time.Minute)
	identityID := uuid.New()

	identity := row{
		ID:        identityID,
		Status:    "disabled",
		CreatedAt: now,
		UpdatedAt: disabledAt,
	}
	emails := []emailRow{
		{
			ID:         uuid.New(),
			IdentityID: identityID,
			Email:      "person@example.com",
			IsPrimary:  true,
			VerifiedAt: &verifiedAt,
			CreatedAt:  now,
			UpdatedAt:  verifiedAt,
		},
	}
	phones := []phoneRow{
		{
			ID:         uuid.New(),
			IdentityID: identityID,
			Phone:      "+79990000000",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	accounts := []externalAccountRow{
		{
			ID:                  uuid.New(),
			IdentityID:          identityID,
			Provider:            "oidc",
			Subject:             "subject-1",
			LinkedAt:            now,
			LastAuthenticatedAt: &lastAuthenticatedAt,
			DisabledAt:          &disabledAt,
		},
	}

	entity := toDomain(identity, emails, phones, accounts)

	require.Equal(t, identityID, entity.ID.UUID())
	require.Equal(t, emails[0].Email, string(entity.Emails[0].Address))
	require.Equal(t, phones[0].Phone, string(entity.Phones[0].Number))
	require.Equal(t, accounts[0].Subject, string(entity.ExternalAccounts[0].Subject))
	require.NotSame(t, emails[0].VerifiedAt, entity.Emails[0].VerifiedAt)
	require.NotSame(t, accounts[0].DisabledAt, entity.ExternalAccounts[0].DisabledAt)
}
