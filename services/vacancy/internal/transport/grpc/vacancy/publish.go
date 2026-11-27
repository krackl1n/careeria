package vacancy

import (
	"context"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy/converter"
)

func (h *Handler) PublishVacancy(ctx context.Context, req *pb.PublishVacancyRequest) (*pb.PublishVacancyResponse, error) {
	id, err := converter.ParseVacancyID(req.VacancyId)
	if err != nil {
		return nil, err
	}
	result, err := h.service.Publish(ctx, id, req.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	return converter.ToPublishResponse(result), nil
}
