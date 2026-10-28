package authentication

import (
	"context"

	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
)

type contextLoader interface {
	LoadContext(context.Context, *transactiondomain.AuthorizationTransaction) (authenticationusecase.ContextOutput, error)
}

func contextForTransaction(
	ctx context.Context,
	loader contextLoader,
	transaction *transactiondomain.AuthorizationTransaction,
) (*identityv1.AuthenticationContext, error) {
	output, err := loader.LoadContext(ctx, transaction)
	if err != nil {
		return nil, err
	}
	return ToAuthenticationContext(output)
}
