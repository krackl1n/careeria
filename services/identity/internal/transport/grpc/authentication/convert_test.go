package authentication

import (
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
	"github.com/stretchr/testify/require"
)

func TestToAuthenticationContextConvertsEmailVerification(t *testing.T) {
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	output := authenticationContextFixture(t, now, transactiondomain.NextStepEmailVerification)
	reference, err := transactiondomain.NewVerificationReference(
		transactiondomain.VerificationChannelEmail,
		output.Identity.Emails[0].ID.UUID(),
		uuid.New(),
		now.Add(5*time.Minute),
	)
	require.NoError(t, err)
	output.AuthorizationTransaction.Verification = &reference

	result, err := ToAuthenticationContext(output)

	require.NoError(t, err)
	require.NoError(t, result.ValidateAll())
	require.Equal(t, "u***@example.com", result.GetNextStep().GetEmailVerification().GetMaskedEmail())
	require.Equal(t, "user@example.com", result.GetIdentity().GetEmail())
}

func TestToAuthenticationContextConvertsMFAEnrollment(t *testing.T) {
	output := authenticationContextFixture(t, time.Now().UTC(), transactiondomain.NextStepMFAEnrollment)

	result, err := ToAuthenticationContext(output)

	require.NoError(t, err)
	require.NoError(t, result.ValidateAll())
	require.Equal(t, []string{"totp"}, result.GetNextStep().GetMfaEnrollment().GetAvailableMethods())
}

func authenticationContextFixture(
	t *testing.T,
	now time.Time,
	nextStep transactiondomain.NextStep,
) authenticationusecase.ContextOutput {
	t.Helper()
	identityID := identitydomain.ID(uuid.New())
	email, err := identitydomain.NewEmailAddress("user@example.com")
	require.NoError(t, err)
	identity, err := identitydomain.NewWithEmail(identityID, identitydomain.EmailID(uuid.New()), email, now)
	require.NoError(t, err)
	profile, err := identityprofiledomain.New(identityID, "Иванов", "Иван", nil, now)
	require.NoError(t, err)
	clientID, err := oauthclientdomain.NewClientID("web")
	require.NoError(t, err)
	return authenticationusecase.ContextOutput{
		AuthorizationTransaction: &transactiondomain.AuthorizationTransaction{
			ID:              transactiondomain.ID(uuid.New()),
			RequestedScopes: []oauthdomain.Scope{"openid"},
			NextStep:        nextStep,
			IdentityID:      &identityID,
			ExpiresAt:       now.Add(10 * time.Minute),
		},
		OAuthClient:         &oauthclientdomain.OAuthClient{ClientID: clientID, Name: "RVSpace Web"},
		Identity:            identity,
		IdentityProfile:     profile,
		AvailableMFAMethods: []authenticationdomain.Method{authenticationdomain.MethodTOTP},
	}
}
