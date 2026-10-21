package offer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	eventdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/event"
	offerdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/offer"
	"github.com/stretchr/testify/require"
)

type mockOfferRepo struct {
	offers map[uuid.UUID]*offerdomain.JobOffer
}

func newMockOfferRepo() *mockOfferRepo {
	return &mockOfferRepo{offers: make(map[uuid.UUID]*offerdomain.JobOffer)}
}

func (m *mockOfferRepo) Create(_ context.Context, of *offerdomain.JobOffer) error {
	m.offers[of.ID] = of
	return nil
}

func (m *mockOfferRepo) Update(_ context.Context, of *offerdomain.JobOffer) error {
	m.offers[of.ID] = of
	return nil
}

func (m *mockOfferRepo) GetByID(_ context.Context, id uuid.UUID) (*offerdomain.JobOffer, error) {
	of, ok := m.offers[id]
	if !ok {
		return nil, offerdomain.ErrNotFound
	}
	return of, nil
}

func (m *mockOfferRepo) GetByApplicationID(_ context.Context, appID uuid.UUID) (*offerdomain.JobOffer, error) {
	for _, of := range m.offers {
		if of.ApplicationID == appID {
			return of, nil
		}
	}
	return nil, offerdomain.ErrNotFound
}

type mockAppCoordinator struct {
	app              *applicationdomain.JobApplication
	advanceCalled    bool
	revertCalled     bool
	lastRevertReason string
}

func (m *mockAppCoordinator) GetApplication(_ context.Context, _ uuid.UUID) (*applicationdomain.JobApplication, error) {
	if m.app == nil {
		return nil, applicationdomain.ErrNotFound
	}
	return m.app, nil
}

func (m *mockAppCoordinator) AdvanceToOfferAccepted(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
	m.advanceCalled = true
	return nil
}

func (m *mockAppCoordinator) RevertToInterview(_ context.Context, _ uuid.UUID, _ uuid.UUID, reason string, _ time.Time) error {
	m.revertCalled = true
	m.lastRevertReason = reason
	return nil
}

type mockOutboxRepo struct {
	messages []eventdomain.Message
}

func (m *mockOutboxRepo) Save(_ context.Context, msg eventdomain.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

type mockAuthPort struct{}

func (m *mockAuthPort) CanManageOffer(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockAuthPort) CanRespondToOffer(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

type mockTxManager struct{}

func (m *mockTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fixedIDGen struct {
	id uuid.UUID
}

func (g fixedIDGen) New() uuid.UUID {
	return g.id
}

func TestOfferService(t *testing.T) {
	ctx := context.Background()
	offerRepo := newMockOfferRepo()
	outboxRepo := &mockOutboxRepo{}
	authPort := &mockAuthPort{}
	txMgr := &mockTxManager{}

	now := time.Now().UTC()
	clock := fixedClock{now: now}
	offerID := uuid.New()
	idGen := fixedIDGen{id: offerID}
	eventFactory := eventdomain.NewFactory(idGen)

	appID := uuid.New()
	candidateID := uuid.New()
	actorID := uuid.New()

	app, err := applicationdomain.New(appID, uuid.New(), candidateID, actorID, now)
	require.NoError(t, err)

	_ = app.ChangeStage(applicationdomain.StageScreening, actorID, "screening", now)
	_ = app.MarkInterview(actorID, "interview", now)
	_ = app.MarkOfferPending(actorID, now)

	appCoord := &mockAppCoordinator{app: app}

	svc := NewService(offerRepo, appCoord, outboxRepo, authPort, txMgr, eventFactory, clock, idGen)

	expiresAt := now.Add(72 * time.Hour)

	id, err := svc.CreateOffer(ctx, CreateOfferInput{
		ApplicationID: appID,
		Compensation:  350000,
		Currency:      offerdomain.CurrencyRUB,
		Grade:         "Principal Engineer",
		ExpiresAt:     expiresAt,
		ActorID:       actorID,
	})
	require.NoError(t, err)
	require.Equal(t, offerID, id)
	require.Len(t, outboxRepo.messages, 1)

	err = svc.SendOffer(ctx, SendOfferInput{
		OfferID: offerID,
		ActorID: actorID,
	})
	require.NoError(t, err)
	require.Len(t, outboxRepo.messages, 2)

	err = svc.AcceptOffer(ctx, AcceptOfferInput{
		OfferID: offerID,
		ActorID: candidateID,
	})
	require.NoError(t, err)
	require.True(t, appCoord.advanceCalled)
	require.Len(t, outboxRepo.messages, 3)

	of, err := svc.GetOffer(ctx, offerID, actorID)
	require.NoError(t, err)
	require.Equal(t, offerdomain.StatusAccepted, of.Status)
}

