package grpc

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryPrincipal reads the authenticated user supplied by the gateway.
func UnaryPrincipal(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	required := true
	switch request := req.(type) {
	case *pb.GetVacancyByIDRequest:
		required = request.View == pb.VacancyView_VACANCY_VIEW_COMPANY
	case *pb.ListCompanyVacanciesRequest:
		required = request.View == pb.VacancyView_VACANCY_VIEW_COMPANY
	}
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("x-principal-sub")
	if len(values) == 0 && !required {
		return handler(ctx, req)
	}
	if len(values) != 1 {
		return nil, status.Error(codes.Unauthenticated, "authenticated principal is required")
	}
	id, err := uuid.Parse(values[0])
	if err != nil || id == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authenticated principal is invalid")
	}
	return handler(ctxmeta.WithPrincipal(ctx, ctxmeta.Principal{SubjectID: id.String()}), req)
}
