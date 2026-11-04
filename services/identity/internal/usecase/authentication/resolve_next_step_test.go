package authentication

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	"github.com/stretchr/testify/require"
)

func TestResolveNextStepLoadsFactsOnlyAfterLocalRequirements(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	identityID := identitydomain.ID(uuid.New())
	transaction := transactionForNextStepTest()
	reader := &factsReaderStub{facts: factsForNextStepTest(t)}
	service := &Service{authenticationFlowFacts: reader}

	decision, err := service.resolveNextStep(
		t.Context(),
		transaction,
		identityID,
		flowdomain.Requirements{PasswordChangeRequired: true},
		now,
	)
	require.NoError(t, err)
	require.Equal(t, transactiondomain.NextStepPasswordChange, decision.NextStep)
	require.Zero(t, reader.calls)

	decision, err = service.resolveNextStep(
		t.Context(),
		transaction,
		identityID,
		flowdomain.Requirements{ContactVerification: flowdomain.ContactVerificationEmail},
		now,
	)
	require.NoError(t, err)
	require.Equal(t, transactiondomain.NextStepEmailVerification, decision.NextStep)
	require.Zero(t, reader.calls)
}

func TestResolveNextStepLoadsSingleFactsSnapshotForMFA(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	identityID := identitydomain.ID(uuid.New())
	transaction := transactionForNextStepTest()
	reader := &factsReaderStub{facts: factsForNextStepTest(t)}
	service := &Service{authenticationFlowFacts: reader}

	decision, err := service.resolveNextStep(
		t.Context(),
		transaction,
		identityID,
		flowdomain.Requirements{},
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	require.Equal(t, transactiondomain.NextStepMFAEnrollment, decision.NextStep)
	require.Equal(t, 1, reader.calls)
	require.Equal(t, identityID, reader.identityID)
	require.Equal(t, transaction.OAuthClientID, reader.oauthClientID)
	require.Equal(t, transaction.RequestedScopes, reader.requestedScopes)
}

type factsReaderStub struct {
	facts           flowdomain.Facts
	calls           int
	identityID      identitydomain.ID
	oauthClientID   oauthclientdomain.ID
	requestedScopes []oauthdomain.Scope
}

func (reader *factsReaderStub) Get(
	_ context.Context,
	identityID identitydomain.ID,
	oauthClientID oauthclientdomain.ID,
	requestedScopes []oauthdomain.Scope,
	_ time.Time,
) (flowdomain.Facts, error) {
	reader.calls++
	reader.identityID = identityID
	reader.oauthClientID = oauthClientID
	reader.requestedScopes = append([]oauthdomain.Scope(nil), requestedScopes...)
	return reader.facts, nil
}

func transactionForNextStepTest() *transactiondomain.AuthorizationTransaction {
	return &transactiondomain.AuthorizationTransaction{
		OAuthClientID:   oauthclientdomain.ID(uuid.New()),
		RequestedScopes: []oauthdomain.Scope{"openid", "profile"},
		NextStep:        transactiondomain.NextStepChooseAuthenticationMethod,
	}
}

func factsForNextStepTest(t *testing.T) flowdomain.Facts {
	t.Helper()
	facts, err := flowdomain.NewFacts(nil, oauthclientdomain.ConsentModeRequired, false)
	require.NoError(t, err)
	return facts
}
