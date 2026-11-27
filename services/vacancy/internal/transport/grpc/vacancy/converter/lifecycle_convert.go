package converter

import (
	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ParseVacancyID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid vacancy id")
	}
	return id, nil
}

func ToUpdateDetailsResponse(entity *domain.Vacancy) *pb.UpdateVacancyDetailsResponse {
	state := entity.Snapshot()
	return &pb.UpdateVacancyDetailsResponse{VacancyId: state.ID.String(), Version: state.Version, UpdatedAt: timestamppb.New(state.UpdatedAt)}
}

func ToPublishResponse(entity *domain.Vacancy) *pb.PublishVacancyResponse {
	state := entity.Snapshot()
	return &pb.PublishVacancyResponse{VacancyId: state.ID.String(), Status: statusValue(state.Status), Version: state.Version, PublishedAt: timestamppb.New(state.PublishedAt)}
}

func ToArchiveResponse(entity *domain.Vacancy) *pb.ArchiveVacancyResponse {
	state := entity.Snapshot()
	return &pb.ArchiveVacancyResponse{VacancyId: state.ID.String(), Status: statusValue(state.Status), Version: state.Version, UpdatedAt: timestamppb.New(state.UpdatedAt)}
}

func ToRestoreResponse(entity *domain.Vacancy) *pb.RestoreVacancyResponse {
	state := entity.Snapshot()
	return &pb.RestoreVacancyResponse{VacancyId: state.ID.String(), Status: statusValue(state.Status), Version: state.Version, PublishedAt: timestamppb.New(state.PublishedAt)}
}

func ToCloseResponse(entity *domain.Vacancy) *pb.CloseVacancyResponse {
	state := entity.Snapshot()
	return &pb.CloseVacancyResponse{VacancyId: state.ID.String(), Status: statusValue(state.Status), Version: state.Version, ClosedAt: timestamppb.New(state.ClosedAt)}
}
