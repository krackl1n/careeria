package vacancy

import (
	"context"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) CreateDraftVacancy(ctx context.Context, req *pb.CreateDraftVacancyRequest) (*pb.CreateDraftVacancyResponse, error) {
	input, err := converter.ToCreateDraftInput(ctx, req)
	if err != nil {
		return nil, err
	}

	entity, err := h.service.CreateDraft(ctx, input)
	if err != nil {
		return nil, err
	}

	return converter.ToCreateDraftResponse(entity), nil
}
