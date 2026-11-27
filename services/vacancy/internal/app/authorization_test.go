package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/ctxmeta"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

type authorizerStub struct {
	access domain.Access
	err    error
}

func (s *authorizerStub) Require(_ context.Context, access domain.Access) error {
	s.access = access
	return s.err
}

func TestRequire(t *testing.T) {
	fga := &authorizerStub{}
	service := &authorizedVacancyService{authorizer: fga}
	company, resource, user := uuid.New(), uuid.New(), uuid.New()
	ctx := ctxmeta.WithPrincipal(context.Background(), ctxmeta.Principal{SubjectID: user.String()})
	if err := service.require(ctx, company, "vacancy", resource, "can_publish"); err != nil {
		t.Fatal(err)
	}
	if fga.access.UserID != user || fga.access.ResourceID != resource || fga.access.Resource != "vacancy" ||
		fga.access.CompanyID != company || fga.access.Permission != "can_publish" {
		t.Fatalf("wrong authorization request: %+v", fga.access)
	}
	fga.err = domain.ErrPermissionDenied
	if err := service.require(ctx, company, "company", company, "can_create_vacancy"); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("deny: %v", err)
	}
	fga.err = errors.New("unavailable")
	if err := service.require(ctx, company, "company", company, "can_create_vacancy"); !errors.Is(err, domain.ErrAuthorizationUnavailable) {
		t.Fatalf("failure: %v", err)
	}
	if err := service.require(context.Background(), company, "company", company, "can_create_vacancy"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("missing principal: %v", err)
	}
}
