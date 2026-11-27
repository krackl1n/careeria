package openfga

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/authz"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

type authorizerStub struct {
	access authz.Access
	err    error
}

func (s *authorizerStub) Check(context.Context, authz.Access) (authz.Decision, error) {
	panic("unexpected Check")
}
func (s *authorizerStub) Require(_ context.Context, access authz.Access) error {
	s.access = access
	return s.err
}

func TestRequire(t *testing.T) {
	client := &authorizerStub{}
	adapter, err := NewAuthorizer(client)
	if err != nil {
		t.Fatal(err)
	}
	access := domain.Access{UserID: uuid.New(), CompanyID: uuid.New(), ResourceID: uuid.New(), Resource: "vacancy", Permission: "can_publish"}
	if err := adapter.Require(context.Background(), access); err != nil {
		t.Fatal(err)
	}
	if client.access.Actor.Type != "user" || client.access.Actor.ID != access.UserID.String() ||
		client.access.Tenant.ID != access.CompanyID.String() || client.access.Resource.String() != "vacancy:"+access.ResourceID.String() ||
		client.access.Permission != access.Permission {
		t.Fatalf("unexpected OpenFGA access: %+v", client.access)
	}
	client.err = authz.ErrDenied
	if err := adapter.Require(context.Background(), access); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("denied: %v", err)
	}
	failure := errors.New("unavailable")
	client.err = failure
	if err := adapter.Require(context.Background(), access); !errors.Is(err, failure) {
		t.Fatalf("failure: %v", err)
	}
	access.UserID = uuid.Nil
	if err := adapter.Require(context.Background(), access); err == nil {
		t.Fatal("expected invalid ID error")
	}
}
