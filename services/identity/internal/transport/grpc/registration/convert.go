package registration

import (
	"github.com/google/uuid"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
)

func toRegisterIdentityInput(request *identityv1.RegisterIdentityRequest) authenticationusecase.RegisterIdentityInput {
	transactionID, _ := uuid.Parse(request.GetAuthorizationTransactionId())
	input := authenticationusecase.RegisterIdentityInput{
		AuthorizationTransactionID: transactionID,
		Password:                   request.GetPassword(),
	}
	switch contact := request.GetContact().(type) {
	case *identityv1.RegisterIdentityRequest_Email:
		input.Email = &contact.Email
	case *identityv1.RegisterIdentityRequest_PhoneNumber:
		input.PhoneNumber = &contact.PhoneNumber
	}
	if request.Profile != nil {
		input.Profile = &authenticationusecase.RegistrationProfileInput{
			FamilyName: request.Profile.GetFamilyName(),
			GivenName:  request.Profile.GetGivenName(),
			MiddleName: request.Profile.MiddleName,
		}
	}
	return input
}
