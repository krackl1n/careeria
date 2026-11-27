package openfga

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/authz"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

type Client interface {
	Require(context.Context, authz.Access) error
}

type Authorizer struct {
	client Client
}

func NewAuthorizer(client Client) (*Authorizer, error) {
	if client == nil {
		return nil, errors.New("openfga client is required")
	}
	return &Authorizer{client: client}, nil
}

func (a *Authorizer) Require(ctx context.Context, access domain.Access) error {
	if access.UserID == uuid.Nil || access.CompanyID == uuid.Nil || access.ResourceID == uuid.Nil {
		return errors.New("authorization user, company and resource IDs are required")
	}
	err := a.client.Require(ctx, authz.Access{
		Actor:      authz.Actor{Type: "user", ID: access.UserID.String()},
		Tenant:     authz.Tenant{ID: access.CompanyID.String()},
		Resource:   authz.Resource(access.Resource, access.ResourceID.String()),
		Permission: access.Permission,
	})
	if errors.Is(err, authz.ErrDenied) {
		return domain.ErrPermissionDenied
	}
	if err != nil {
		return fmt.Errorf("check OpenFGA permission: %w", err)
	}
	return nil
}
