package identity

import (
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	"google.golang.org/grpc"
)

type Clients struct {
	OAuth          identityv1.OAuthServiceClient
	Authentication identityv1.AuthenticationServiceClient
	Session        identityv1.IdentitySessionServiceClient
	Metadata       identityv1.IdentityMetadataServiceClient
	Registration   identityv1.RegistrationServiceClient
}

func New(connection grpc.ClientConnInterface) *Clients {
	return &Clients{
		OAuth:          identityv1.NewOAuthServiceClient(connection),
		Authentication: identityv1.NewAuthenticationServiceClient(connection),
		Session:        identityv1.NewIdentitySessionServiceClient(connection),
		Metadata:       identityv1.NewIdentityMetadataServiceClient(connection),
		Registration:   identityv1.NewRegistrationServiceClient(connection),
	}
}
