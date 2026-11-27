package vacancy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	usecase "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type commandStub struct {
	Service
	patch usecase.DetailsPatch
}

func (s *commandStub) UpdateDetails(_ context.Context, id uuid.UUID, _ int64, patch usecase.DetailsPatch) (*domain.Vacancy, error) {
	s.patch = patch
	return domain.NewDraft(id, uuid.New(), uuid.New(), domain.Details{Title: "Engineer", WorkFormat: domain.WorkRemote}, time.Now())
}

func TestUpdateMaskAndCompensation(t *testing.T) {
	commands := &commandStub{}
	handler := NewHandler(commands)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-principal-sub", uuid.NewString()))
	zero := "0.00"
	req := &pb.UpdateVacancyDetailsRequest{VacancyId: uuid.NewString(), ExpectedVersion: 1,
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"location", "compensation", "skills"}},
		Compensation: &pb.Compensation{Minimum: &zero}}
	_, err := handler.UpdateVacancyDetails(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !commands.patch.Location.Set || commands.patch.Location.Value != nil || !commands.patch.Skills.Set ||
		commands.patch.Compensation.Value == nil || *commands.patch.Compensation.Value.Minimum != 0 {
		t.Fatal("explicit clearing or zero amount lost")
	}
	req.UpdateMask.Paths = []string{"title"}
	if _, err := handler.UpdateVacancyDetails(ctx, req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing title: %v", err)
	}
	req.UpdateMask.Paths = []string{"unknown"}
	if _, err := handler.UpdateVacancyDetails(ctx, req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unknown field: %v", err)
	}
}
