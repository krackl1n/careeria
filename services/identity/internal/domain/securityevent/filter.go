package securityevent

import (
	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type Filter struct {
	IdentityID    *identitydomain.ID
	CorrelationID *uuid.UUID
	Limit         int
	Offset        int
}
