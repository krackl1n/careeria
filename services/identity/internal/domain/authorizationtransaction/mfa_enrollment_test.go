package authorizationtransaction

import (
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationTransactionSkipsMFAEnrollment(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	identityID := identitydomain.ID(uuid.New())
	transaction := &AuthorizationTransaction{
		Status:               StatusPending,
		NextStep:             NextStepMFAEnrollment,
		IdentityID:           &identityID,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword},
		ExpiresAt:            now.Add(10 * time.Minute),
	}

	require.NoError(t, transaction.SkipMFAEnrollment(now))
	require.Equal(t, NextStepConsent, transaction.NextStep)
	require.Equal(t, authenticationdomain.LevelSingleFactor, *transaction.AuthenticationLevel)
	require.Equal(t, now, *transaction.AuthenticatedAt)
}

func TestAuthorizationTransactionRejectsSkippingAnotherStep(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	identityID := identitydomain.ID(uuid.New())
	transaction := &AuthorizationTransaction{
		Status:               StatusPending,
		NextStep:             NextStepMFA,
		IdentityID:           &identityID,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword},
		ExpiresAt:            now.Add(10 * time.Minute),
	}

	err := transaction.SkipMFAEnrollment(now)

	require.ErrorIs(t, err, ErrInvalidStep)
	require.Equal(t, NextStepMFA, transaction.NextStep)
}
