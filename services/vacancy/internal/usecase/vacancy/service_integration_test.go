package vacancy_test

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/services/vacancy/internal/domain/event"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	"github.com/krackl1n/careeria/services/vacancy/internal/repository/outbox"
	repository "github.com/krackl1n/careeria/services/vacancy/internal/repository/vacancy"
	uc "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	query "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"github.com/krackl1n/careeria/txmanager"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type ids struct{}

func (ids) New() uuid.UUID { return uuid.New() }

type failingOutbox struct{}

func (failingOutbox) Create(context.Context, event.Message) error {
	return errors.New("outbox unavailable")
}

func TestCommandsAndQueries(t *testing.T) {
	host := os.Getenv("VACANCY_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("VACANCY_TEST_POSTGRES_HOST is not set")
	}
	port, err := strconv.Atoi(os.Getenv("VACANCY_TEST_POSTGRES_PORT"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := postgres.DefaultConfig()
	cfg.Connection = postgres.ConnectionConfig{Host: host, Port: port, Database: "vacancy", User: "vacancy", Password: "vacancy", SSLMode: "disable"}
	ctx := context.Background()
	client, err := postgres.New(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repo := repository.NewPostgresRepository(client)
	transaction := txmanager.New[pgx.Tx](client)
	clock := fixedClock{now: time.Now().UTC().Truncate(time.Microsecond)}
	service, err := uc.NewService(repo, outbox.NewPostgresRepository(client), transaction, clock, ids{})
	if err != nil {
		t.Fatal(err)
	}
	input := uc.CreateDraftInput{
		RequestID: uuid.New(), CompanyID: uuid.New(), AuthorID: uuid.New(),
		Details: domain.Details{Title: "Engineer", Description: strings.Repeat("a", 50), WorkFormat: domain.WorkRemote, Skills: []domain.SkillRequirement{{Key: "go", DisplayName: "Go"}}},
	}
	created, err := service.CreateDraft(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := service.CreateDraft(ctx, input)
	if err != nil || repeated.Snapshot().ID != created.Snapshot().ID || !repeated.Snapshot().CreatedAt.Equal(created.Snapshot().CreatedAt) {
		t.Fatalf("idempotent create: %+v %v", repeated, err)
	}
	changed := input
	changed.Details.Title = "Another title"
	if _, err := service.CreateDraft(ctx, changed); !errors.Is(err, uc.ErrIdempotencyConflict) {
		t.Fatalf("conflicting create: %v", err)
	}

	broken, err := uc.NewService(repo, failingOutbox{}, transaction, clock, ids{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broken.Publish(ctx, created.Snapshot().ID, 1); err == nil {
		t.Fatal("expected outbox failure")
	}
	entity, err := repo.Get(ctx, created.Snapshot().ID)
	if err != nil || entity.Snapshot().Version != 1 || entity.Snapshot().Status != domain.StatusDraft {
		t.Fatalf("rollback failed: %v", err)
	}
	published, err := service.Publish(ctx, created.Snapshot().ID, 1)
	if err != nil || published.Snapshot().Version != 2 {
		t.Fatalf("publish: %v", err)
	}
	if _, err := service.Publish(ctx, created.Snapshot().ID, 1); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	if _, err := service.UpdateDetails(ctx, created.Snapshot().ID, 2, uc.DetailsPatch{Skills: uc.Field[[]domain.SkillRequirement]{Set: true}}); !errors.Is(err, domain.ErrNotPublishable) {
		t.Fatalf("invalid published edit: %v", err)
	}
	queries, err := query.NewService(repo)
	if err != nil {
		t.Fatal(err)
	}
	card, err := queries.Get(ctx, created.Snapshot().ID, query.ViewPublic)
	if err != nil || card == nil || card.Snapshot().ID != created.Snapshot().ID {
		t.Fatalf("public card: %v", err)
	}
	page, err := queries.ListCompany(ctx, query.ListInput{CompanyID: input.CompanyID, View: query.ViewPublic, PageSize: 1})
	if err != nil || len(page.Vacancies) != 1 {
		t.Fatalf("public page: %v", err)
	}

	secondInput := input
	secondInput.RequestID = uuid.New()
	second, err := service.CreateDraft(ctx, secondInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish(ctx, second.Snapshot().ID, 1); err != nil {
		t.Fatal(err)
	}
	page, err = queries.ListCompany(ctx, query.ListInput{CompanyID: input.CompanyID, View: query.ViewPublic, PageSize: 1})
	if err != nil || len(page.Vacancies) != 1 || page.Next == nil {
		t.Fatalf("first page: %v", err)
	}
	next, err := queries.ListCompany(ctx, query.ListInput{CompanyID: input.CompanyID, View: query.ViewPublic, PageSize: 1, Cursor: page.Next})
	if err != nil || len(next.Vacancies) != 1 || next.Next != nil || next.Vacancies[0].Snapshot().ID == page.Vacancies[0].Snapshot().ID {
		t.Fatalf("next page: %v", err)
	}
	if _, err := service.Archive(ctx, created.Snapshot().ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.Get(ctx, created.Snapshot().ID, query.ViewPublic); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("archived publicly visible: %v", err)
	}
	if _, err := service.Restore(ctx, created.Snapshot().ID, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Close(ctx, created.Snapshot().ID, 4); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := client.Pool().QueryRow(ctx, "select count(*) from outbox_messages where aggregate_id=$1", created.Snapshot().ID).Scan(&count); err != nil || count != 5 {
		t.Fatalf("events count %d: %v", count, err)
	}
}
