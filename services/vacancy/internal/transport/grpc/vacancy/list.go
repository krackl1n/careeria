package vacancy

import (
	"context"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) ListCompanyVacancies(ctx context.Context, req *pb.ListCompanyVacanciesRequest) (*pb.ListCompanyVacanciesResponse, error) {
	input, err := converter.ToListCompanyInput(req)
	if err != nil {
		return nil, err
	}

	result, err := h.service.ListCompany(ctx, input)
	if err != nil {
		return nil, err
	}

	return converter.ToListCompanyResponse(result, input)
}
