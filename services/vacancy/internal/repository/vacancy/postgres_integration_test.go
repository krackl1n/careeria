package vacancy

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/postgres"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func TestPostgresRepository(t *testing.T) {
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
	repo := NewPostgresRepository(client)
	now := time.Now().UTC().Truncate(time.Microsecond)
	company := uuid.New()
	details := domain.Details{Title: "Engineer", Description: strings.Repeat("a", 50), WorkFormat: domain.WorkRemote}
	entity, err := domain.NewDraft(uuid.New(), company, uuid.New(), details, now)
	if err != nil {
		t.Fatal(err)
	}
	requestID, requestHash := uuid.New(), "hash"
	if err := repo.Create(ctx, entity, requestID, requestHash); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, entity, requestID, requestHash); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	stored, hash, err := repo.GetByCreateRequest(ctx, requestID)
	if err != nil || hash != requestHash || stored.Snapshot().ID != entity.Snapshot().ID {
		t.Fatalf("get by request: %v", err)
	}
	details.Skills = []domain.SkillRequirement{{Key: "go", DisplayName: "Go"}}
	min := domain.Amount(12345)
	details.Compensation = &domain.Compensation{Minimum: &min, Currency: "USD", Period: "MONTH"}
	if err := entity.UpdateDetails(1, details, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, entity, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, entity, 1); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale update: %v", err)
	}
	stored, err = repo.Get(ctx, entity.Snapshot().ID)
	if err != nil {
		t.Fatal(err)
	}
	if *stored.Snapshot().Details.Compensation.Minimum != min || len(stored.Snapshot().Details.Skills) != 1 {
		t.Fatal("compensation or skill lost")
	}
	if err := entity.Publish(2, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, entity, 2); err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListCompany(ctx, company, true, 1, nil)
	if err != nil || len(page.Vacancies) != 1 {
		t.Fatalf("public page: %v", err)
	}
	details.Skills = []domain.SkillRequirement{{Key: "go", DisplayName: "Golang"}, {Key: "sql", DisplayName: "SQL"}}
	if err := entity.UpdateDetails(3, details, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, entity, 3); err != nil {
		t.Fatal(err)
	}
	details.Skills = details.Skills[1:]
	if err := entity.UpdateDetails(4, details, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, entity, 4); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.Get(ctx, entity.Snapshot().ID)
	if err != nil || len(stored.Snapshot().Details.Skills) != 1 || stored.Snapshot().Details.Skills[0].Key != "sql" {
		t.Fatalf("replace skills: %v", err)
	}
}
