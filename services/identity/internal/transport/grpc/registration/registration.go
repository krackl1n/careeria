package registration

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	authenticationtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/authentication"
)

func (h *Handler) RegisterIdentity(
	ctx context.Context,
	request *identityv1.RegisterIdentityRequest,
) (*identityv1.RegisterIdentityResponse, error) {
	output, err := h.registration.RegisterIdentity(ctx, toRegisterIdentityInput(request))
	if err != nil {
		return nil, err
	}
	contextOutput, err := h.registration.LoadContext(ctx, output.AuthorizationTransaction)
	if err != nil {
		return nil, err
	}
	contextValue, err := authenticationtransport.ToAuthenticationContext(contextOutput)
	if err != nil {
		return nil, err
	}
	return &identityv1.RegisterIdentityResponse{Context: contextValue}, nil
}
