package registration

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
)

type Handler struct {
	identityv1.UnimplementedRegistrationServiceServer

	registration Service
}

type Service interface {
	RegisterIdentity(context.Context, authenticationusecase.RegisterIdentityInput) (authenticationusecase.RegisterIdentityOutput, error)
	LoadContext(context.Context, *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error)
}

var _ identityv1.RegistrationServiceServer = (*Handler)(nil)

func NewHandler(registration Service) *Handler {
	return &Handler{registration: registration}
}
