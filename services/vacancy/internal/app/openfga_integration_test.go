package app

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	openfga "github.com/krackl1n/careeria/authz/openfga/grpc"
	"github.com/krackl1n/careeria/ctxmeta"
	adapter "github.com/krackl1n/careeria/services/vacancy/internal/integrations/openfga"
	fgapb "github.com/openfga/api/proto/openfga/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type fgaServer struct {
	fgapb.UnimplementedOpenFGAServiceServer
	calls chan *fgapb.CheckRequest
}

func (s *fgaServer) Check(_ context.Context, req *fgapb.CheckRequest) (*fgapb.CheckResponse, error) {
	s.calls <- req
	return &fgapb.CheckResponse{Allowed: true}, nil
}

func TestOpenFGACheck(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	stub := &fgaServer{calls: make(chan *fgapb.CheckRequest, 1)}
	fgapb.RegisterOpenFGAServiceServer(server, stub)
	go server.Serve(listener)
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:///openfga", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) { return listener.Dial() },
	))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client, err := openfga.New(openfga.Config{StoreID: "store", AuthorizationModelID: "model", Timeout: time.Second}, conn)
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := adapter.NewAuthorizer(client)
	if err != nil {
		t.Fatal(err)
	}
	service := &authorizedVacancyService{authorizer: authorization}
	user, company, vacancy := uuid.New(), uuid.New(), uuid.New()
	ctx := ctxmeta.WithPrincipal(context.Background(), ctxmeta.Principal{SubjectID: user.String()})
	if err := service.require(ctx, company, "vacancy", vacancy, "can_publish"); err != nil {
		t.Fatal(err)
	}
	req := <-stub.calls
	if req.StoreId != "store" || req.AuthorizationModelId != "model" || req.TupleKey.User != "user:"+user.String() ||
		req.TupleKey.Relation != "can_publish" || req.TupleKey.Object != "vacancy:"+vacancy.String() ||
		req.Context.GetFields()["tenant_id"].GetStringValue() != company.String() {
		t.Fatalf("unexpected OpenFGA request: %v", req)
	}
}
