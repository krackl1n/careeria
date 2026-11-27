package vacancy

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrPermissionDenied         = errors.New("vacancy access denied")
	ErrUnauthenticated          = errors.New("authenticated principal is required")
	ErrAuthorizationUnavailable = errors.New("authorization service unavailable")
)

type Access struct {
	UserID     uuid.UUID
	CompanyID  uuid.UUID
	ResourceID uuid.UUID
	Resource   string
	Permission string
}

type Authorizer interface {
	Require(context.Context, Access) error
}
