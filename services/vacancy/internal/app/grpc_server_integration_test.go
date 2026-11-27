package app

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	"github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/services/vacancy/internal/config"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	repository "github.com/krackl1n/careeria/services/vacancy/internal/repository/vacancy"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type denyAuthorizer struct{ allow atomic.Bool }

func (a *denyAuthorizer) Require(context.Context, domain.Access) error {
	if a.allow.Load() {
		return nil
	}
	return domain.ErrPermissionDenied
}

func TestGRPCServer(t *testing.T) {
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
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	log := logger.New(logger.Config{Level: "error", Format: logger.FormatJSON, Service: "vacancy"}, nil)
	authorizer := &denyAuthorizer{}
	server, err := newGRPCServer(config.Config{GRPCServer: grpcserver.Config{Addr: addr}}, log, client, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	go server.Start()
	defer server.Shutdown(context.Background())
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := pb.NewVacancyServiceClient(conn)
	query := pb.NewVacancyQueryServiceClient(conn)
	_, err = command.CreateDraftVacancy(ctx, &pb.CreateDraftVacancyRequest{}, grpc.WaitForReady(true))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("protected command validation: %v", err)
	}
	// Valid UUIDs with an invalid nested DTO must be rejected by the validation
	// interceptor before authentication or the handler is reached.
	_, err = command.CreateDraftVacancy(ctx, &pb.CreateDraftVacancyRequest{
		CreateRequestId: uuid.NewString(),
		CompanyId:       uuid.NewString(),
		Details:         &pb.VacancyDetails{Title: "", WorkFormat: pb.WorkFormat_WORK_FORMAT_UNSPECIFIED},
	})
	validationStatus := status.Convert(err)
	if validationStatus.Code() != codes.InvalidArgument {
		t.Fatalf("nested proto validation: %v", err)
	}
	var validationDetails *errdetails.BadRequest
	for _, detail := range validationStatus.Details() {
		if badRequest, ok := detail.(*errdetails.BadRequest); ok {
			validationDetails = badRequest
		}
	}
	if validationDetails == nil || len(validationDetails.FieldViolations) == 0 {
		t.Fatalf("missing validation details: %v", err)
	}

	_, err = query.GetVacancyByID(metadata.AppendToOutgoingContext(ctx, "x-principal-sub", uuid.NewString()), &pb.GetVacancyByIDRequest{VacancyId: uuid.NewString(), View: pb.VacancyView_VACANCY_VIEW_COMPANY})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unknown company vacancy: %v", err)
	}
	_, err = query.GetVacancyByID(ctx, &pb.GetVacancyByIDRequest{VacancyId: "bad"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("validation: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	entity, err := domain.NewDraft(uuid.New(), uuid.New(), uuid.New(), domain.Details{Title: "Engineer", Description: strings.Repeat("a", 50), WorkFormat: domain.WorkRemote, Skills: []domain.SkillRequirement{{Key: "go", DisplayName: "Go"}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewPostgresRepository(client)
	if err := repo.Create(ctx, entity, uuid.New(), "hash"); err != nil {
		t.Fatal(err)
	}
	_, err = query.GetVacancyByID(ctx, &pb.GetVacancyByIDRequest{VacancyId: entity.Snapshot().ID.String()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("draft visibility: %v", err)
	}
	actorCtx := metadata.AppendToOutgoingContext(ctx, "x-principal-sub", uuid.NewString())
	_, err = command.PublishVacancy(actorCtx, &pb.PublishVacancyRequest{VacancyId: entity.Snapshot().ID.String(), ExpectedVersion: 1})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("publish without permission: %v", err)
	}
	authorizer.allow.Store(true)
	_, err = command.PublishVacancy(actorCtx, &pb.PublishVacancyRequest{VacancyId: entity.Snapshot().ID.String(), ExpectedVersion: 1})
	if err != nil {
		t.Fatalf("authorized publish: %v", err)
	}
	res, err := query.GetVacancyByID(ctx, &pb.GetVacancyByIDRequest{VacancyId: entity.Snapshot().ID.String()})
	if err != nil || res.GetVacancy().GetPublicVacancy().GetDetails().GetTitle() != "Engineer" {
		t.Fatalf("public vacancy: %v", err)
	}
}
