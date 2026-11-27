package vacancy

import (
	"context"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) CloseVacancy(ctx context.Context, req *pb.CloseVacancyRequest) (*pb.CloseVacancyResponse, error) {
	id, err := converter.ParseVacancyID(req.VacancyId)
	if err != nil {
		return nil, err
	}
	result, err := h.service.Close(ctx, id, req.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	return converter.ToCloseResponse(result), nil
}
