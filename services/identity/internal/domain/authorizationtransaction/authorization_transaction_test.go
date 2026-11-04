package authorizationtransaction

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationTransactionCanBeCompleted(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())

	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStepMFA,
		now.Add(time.Minute),
	))
	require.NoError(t, transaction.CompleteMFA(
		identityID,
		authenticationdomain.MethodTOTP,
		now.Add(2*time.Minute),
	))
	require.NoError(t, transaction.AuthorizeScopes(
		[]oauthdomain.Scope{"openid"},
		AuthorizationSourceUserConsent,
		now.Add(3*time.Minute),
	))
	require.NoError(t, transaction.Complete(now.Add(4*time.Minute)))
	require.Equal(t, StatusCompleted, transaction.Status)
	require.Empty(t, transaction.NextStep)
	require.Equal(t, identityID, *transaction.IdentityID)
	require.Equal(t, []oauthdomain.Scope{"openid"}, transaction.GrantedScopes)
}

func TestAuthorizationTransactionRequiresPKCEWhenConfigured(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	_, err := New(
		ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		redirectURIForTest(t),
		[]oauthdomain.Scope{"openid"},
		nil,
		nil,
		true,
		nil,
		now.Add(10*time.Minute),
		now,
	)
	require.ErrorIs(t, err, ErrPKCERequired)
}

func TestAuthorizationTransactionOnlyGrantsRequestedScopes(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identitydomain.ID(uuid.New()),
		authenticationdomain.MethodPassword,
		NextStepConsent,
		now.Add(time.Minute),
	))

	err := transaction.AuthorizeScopes(
		[]oauthdomain.Scope{"admin"},
		AuthorizationSourceUserConsent,
		now.Add(2*time.Minute),
	)
	require.Error(t, err)
	require.Equal(t, StatusPending, transaction.Status)
}

func TestAuthorizationTransactionCannotBeCompletedBeforeAuthentication(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	err := transaction.Complete(now.Add(time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
	require.Equal(t, StatusPending, transaction.Status)
}

func TestCompletedAuthorizationTransactionCannotBeChanged(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identitydomain.ID(uuid.New()),
		authenticationdomain.MethodPassword,
		NextStepConsent,
		now.Add(time.Minute),
	))
	require.NoError(t, transaction.AuthorizeScopes(
		[]oauthdomain.Scope{"openid"},
		AuthorizationSourceTrustedClient,
		now.Add(2*time.Minute),
	))
	require.NoError(t, transaction.Complete(now.Add(3*time.Minute)))

	err := transaction.Complete(now.Add(4 * time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestExpiredAuthorizationTransactionCannotChange(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	err := transaction.CompletePrimaryAuthentication(
		identitydomain.ID(uuid.New()),
		authenticationdomain.MethodPassword,
		NextStepConsent,
		transaction.ExpiresAt,
	)
	require.ErrorIs(t, err, ErrExpired)
	require.Equal(t, StatusPending, transaction.Status)
}

func TestAuthorizationTransactionCanAdvanceThroughIntermediateSteps(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())

	require.Equal(t, NextStepChooseAuthenticationMethod, transaction.NextStep)
	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStepPasswordChange,
		now.Add(time.Minute),
	))
	require.NoError(t, transaction.CompletePasswordChange(NextStepEmailVerification, now.Add(2*time.Minute)))
	require.NoError(t, transaction.BindEmailVerification(
		identitydomain.EmailID(uuid.New()),
		uuid.New(),
		now.Add(8*time.Minute),
		now.Add(2*time.Minute),
	))
	require.NoError(t, transaction.CompleteEmailVerification(NextStepPhoneVerification, now.Add(3*time.Minute)))
	require.Nil(t, transaction.Verification)
	require.NoError(t, transaction.BindPhoneVerification(
		identitydomain.PhoneID(uuid.New()),
		uuid.New(),
		now.Add(8*time.Minute),
		now.Add(3*time.Minute),
	))
	require.NoError(t, transaction.CompletePhoneVerification(NextStepMFA, now.Add(4*time.Minute)))
	require.Nil(t, transaction.Verification)
	require.NoError(t, transaction.CompleteMFA(
		identityID,
		authenticationdomain.MethodTOTP,
		now.Add(5*time.Minute),
	))
	require.Equal(t, authenticationdomain.LevelMultiFactor, *transaction.AuthenticationLevel)
	require.Equal(t, NextStepConsent, transaction.NextStep)
}

func TestAuthorizationTransactionCanCompletePasswordRegistration(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())

	require.NoError(t, transaction.BeginPasswordRegistration(
		identityID,
		NextStepEmailVerification,
		now.Add(time.Minute),
	))
	require.Equal(t, identityID, *transaction.IdentityID)
	require.Equal(t, NextStepEmailVerification, transaction.NextStep)
	require.Equal(t, []authenticationdomain.Method{authenticationdomain.MethodPassword}, transaction.AuthenticatedMethods)
	require.Nil(t, transaction.AuthenticationLevel)
	require.NoError(t, transaction.BindEmailVerification(
		identitydomain.EmailID(uuid.New()),
		uuid.New(),
		now.Add(8*time.Minute),
		now.Add(time.Minute),
	))
	require.NoError(t, transaction.CompleteEmailVerification(NextStepConsent, now.Add(2*time.Minute)))
	require.Equal(t, authenticationdomain.LevelSingleFactor, *transaction.AuthenticationLevel)
	require.Equal(t, NextStepConsent, transaction.NextStep)

	err := transaction.BeginPasswordRegistration(
		identitydomain.ID(uuid.New()),
		NextStepPhoneVerification,
		now.Add(3*time.Minute),
	)
	require.ErrorIs(t, err, ErrInvalidStep)
}

func TestAuthorizationTransactionCanSkipConsentWithRememberedGrant(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())

	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStepConsent,
		now.Add(time.Minute),
	))
	require.NoError(t, transaction.AuthorizeScopes(
		[]oauthdomain.Scope{"openid", "profile"},
		AuthorizationSourceRememberedGrant,
		now.Add(2*time.Minute),
	))
	require.Equal(t, StatusReady, transaction.Status)
	require.Equal(t, NextStepReady, transaction.NextStep)
}

func TestAuthorizationTransactionRejectsUnknownStep(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)

	identityID := identitydomain.ID(uuid.New())
	err := transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStep("unknown"),
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrInvalidStep)
	require.Equal(t, NextStepChooseAuthenticationMethod, transaction.NextStep)
	require.Nil(t, transaction.IdentityID)
	require.Empty(t, transaction.AuthenticatedMethods)
}

func TestAuthorizationTransactionAppliesAuthenticationAndTransitionAtomically(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())
	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStepConsent,
		now.Add(time.Minute),
	))
	require.Equal(t, NextStepConsent, transaction.NextStep)
	require.Equal(t, identityID, *transaction.IdentityID)
	require.Equal(t, []authenticationdomain.Method{authenticationdomain.MethodPassword}, transaction.AuthenticatedMethods)
	require.Equal(t, authenticationdomain.LevelSingleFactor, *transaction.AuthenticationLevel)
	require.Equal(t, now.Add(time.Minute), *transaction.AuthenticatedAt)
}

func TestAuthorizationTransactionRejectsMFACompletionOutsideMFAStep(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)

	err := transaction.CompleteMFA(
		identitydomain.ID(uuid.New()),
		authenticationdomain.MethodTOTP,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrInvalidStep)
	require.Equal(t, NextStepChooseAuthenticationMethod, transaction.NextStep)
	require.Nil(t, transaction.IdentityID)
	require.Empty(t, transaction.AuthenticatedMethods)
}

func TestAuthorizationTransactionDoesNotCompleteRequirementOnInvalidTransition(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	require.NoError(t, transaction.BeginPasswordRegistration(
		identitydomain.ID(uuid.New()),
		NextStepEmailVerification,
		now.Add(time.Minute),
	))

	err := transaction.CompleteEmailVerification(NextStepReady, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrInvalidStep)
	require.Equal(t, NextStepEmailVerification, transaction.NextStep)
	require.Nil(t, transaction.AuthenticationLevel)
	require.Nil(t, transaction.AuthenticatedAt)
}

func TestAuthorizationTransactionCannotAddFactorAfterAuthenticationCompleted(t *testing.T) {
	t.Parallel()

	now := authorizationTransactionTestTime()
	transaction := newAuthorizationTransactionForTest(t, true, now)
	identityID := identitydomain.ID(uuid.New())
	require.NoError(t, transaction.CompletePrimaryAuthentication(
		identityID,
		authenticationdomain.MethodPassword,
		NextStepConsent,
		now.Add(time.Minute),
	))

	err := transaction.CompleteMFA(
		identityID,
		authenticationdomain.MethodTOTP,
		now.Add(2*time.Minute),
	)
	require.ErrorIs(t, err, ErrInvalidStep)
	require.Equal(t, []authenticationdomain.Method{authenticationdomain.MethodPassword}, transaction.AuthenticatedMethods)
}

func newAuthorizationTransactionForTest(t *testing.T, requirePKCE bool, now time.Time) *AuthorizationTransaction {
	t.Helper()

	state := State("oauth-state")
	nonce := openidconnectdomain.Nonce("oidc-nonce")
	var pkce *oauthdomain.PKCE
	if requirePKCE {
		challenge := codeChallengeFor(testCodeVerifier)
		method := "S256"
		var err error
		pkce, err = oauthdomain.NewPKCE(&challenge, &method)
		require.NoError(t, err)
	}
	transaction, err := New(
		ID(uuid.New()),
		oauthclientdomain.ID(uuid.New()),
		redirectURIForTest(t),
		[]oauthdomain.Scope{"openid", "profile"},
		&state,
		&nonce,
		requirePKCE,
		pkce,
		now.Add(10*time.Minute),
		now,
	)
	require.NoError(t, err)
	return transaction
}

func redirectURIForTest(t *testing.T) oauthclientdomain.RedirectURIValue {
	t.Helper()
	redirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/callback")
	require.NoError(t, err)
	return redirectURI
}

func codeChallengeFor(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func authorizationTransactionTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}

const testCodeVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
