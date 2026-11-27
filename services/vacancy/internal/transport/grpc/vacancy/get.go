package vacancy

import (
	"context"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) GetVacancyByID(ctx context.Context, req *pb.GetVacancyByIDRequest) (*pb.GetVacancyByIDResponse, error) {
	id, mode, err := converter.ToGetVacancyInput(req)
	if err != nil {
		return nil, err
	}

	result, err := h.service.Get(ctx, id, mode)
	if err != nil {
		return nil, err
	}

	return converter.ToGetVacancyResponse(result, mode), nil
}
