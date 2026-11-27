package converter

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/ctxmeta"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	usecase "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToCreateDraftInput(ctx context.Context, req *pb.CreateDraftVacancyRequest) (usecase.CreateDraftInput, error) {
	principal, _ := ctxmeta.PrincipalFromContext(ctx)
	author, _ := uuid.Parse(principal.SubjectID)

	requestID, err := uuid.Parse(req.GetCreateRequestId())
	if err != nil {
		return usecase.CreateDraftInput{}, status.Error(codes.InvalidArgument, "invalid request id")
	}
	companyID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return usecase.CreateDraftInput{}, status.Error(codes.InvalidArgument, "invalid company id")
	}
	content, err := details(req.GetDetails())
	if err != nil {
		return usecase.CreateDraftInput{}, err
	}

	return usecase.CreateDraftInput{
		RequestID: requestID,
		CompanyID: companyID,
		AuthorID:  author,
		Details:   content,
	}, nil
}

func ToCreateDraftResponse(entity *domain.Vacancy) *pb.CreateDraftVacancyResponse {
	state := entity.Snapshot()
	return &pb.CreateDraftVacancyResponse{
		VacancyId: state.ID.String(),
		Status:    statusValue(state.Status),
		Version:   state.Version,
		CreatedAt: timestamppb.New(state.CreatedAt),
	}
}
