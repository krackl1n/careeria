package identity

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestConflictError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		constraint string
		expected   error
	}{
		{
			name:       "email",
			constraint: "identity_emails_email_uidx",
			expected:   identitydomain.ErrEmailAlreadyExists,
		},
		{
			name:       "phone",
			constraint: "identity_phones_phone_uidx",
			expected:   identitydomain.ErrPhoneAlreadyExists,
		},
		{
			name:       "external account",
			constraint: "external_identity_accounts_provider_subject_uidx",
			expected:   identitydomain.ErrExternalAccountExists,
		},
		{name: "unmapped", constraint: "identities_pkey"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := &pgconn.PgError{
				Code:           sharedpostgres.CodeUniqueViolation,
				ConstraintName: test.constraint,
			}

			require.Equal(t, test.expected, conflictError(err))
		})
	}
}
