package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryPrincipal(t *testing.T) {
	id := uuid.NewString()
	called := false
	next := func(ctx context.Context, _ any) (any, error) {
		called = true
		principal, ok := ctxmeta.PrincipalFromContext(ctx)
		if !ok || principal.SubjectID != id {
			t.Fatalf("principal: %+v", principal)
		}
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-principal-sub", id))
	if _, err := UnaryPrincipal(ctx, &pb.PublishVacancyRequest{}, &grpc.UnaryServerInfo{}, next); err != nil || !called {
		t.Fatalf("authenticated request: %v", err)
	}
	called = false
	if _, err := UnaryPrincipal(context.Background(), &pb.PublishVacancyRequest{}, &grpc.UnaryServerInfo{}, next); status.Code(err) != codes.Unauthenticated || called {
		t.Fatalf("missing principal: %v", err)
	}
	duplicate := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-principal-sub", id, "x-principal-sub", id))
	if _, err := UnaryPrincipal(duplicate, &pb.PublishVacancyRequest{}, &grpc.UnaryServerInfo{}, next); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("duplicate principal: %v", err)
	}
	called = false
	_, err := UnaryPrincipal(context.Background(), &pb.GetVacancyByIDRequest{}, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) { called = true; return nil, nil })
	if err != nil || !called {
		t.Fatalf("public request: %v", err)
	}
}
