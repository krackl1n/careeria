package vacancy

import (
	"context"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) UpdateVacancyDetails(ctx context.Context, req *pb.UpdateVacancyDetailsRequest) (*pb.UpdateVacancyDetailsResponse, error) {
	id, patch, err := converter.ToUpdateDetailsInput(req)
	if err != nil {
		return nil, err
	}

	result, err := h.service.UpdateDetails(ctx, id, req.ExpectedVersion, patch)
	if err != nil {
		return nil, err
	}
	return converter.ToUpdateDetailsResponse(result), nil
}
