package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedclock "github.com/krackl1n/careeria/clock"
	grpcinterceptor "github.com/krackl1n/careeria/grpc/interceptor"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedidgen "github.com/krackl1n/careeria/idgen"
	sharedkafka "github.com/krackl1n/careeria/kafka"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	sharedruntime "github.com/krackl1n/careeria/runtime"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	kafkaintegration "github.com/krackl1n/careeria/services/identity/internal/integrations/kafka"
	authenticationrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authenticationflow"
	authorizationcoderepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationcode"
	authorizationgrantrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationgrant"
	authorizationtransactionrepository "github.com/krackl1n/careeria/services/identity/internal/repository/authorizationtransaction"
	emailchallengerepository "github.com/krackl1n/careeria/services/identity/internal/repository/emailverificationchallenge"
	identityrepository "github.com/krackl1n/careeria/services/identity/internal/repository/identity"
	identityprofilerepository "github.com/krackl1n/careeria/services/identity/internal/repository/identityprofile"
	oauthclientrepository "github.com/krackl1n/careeria/services/identity/internal/repository/oauthclient"
	outboxrepository "github.com/krackl1n/careeria/services/identity/internal/repository/outbox"
	passwordrepository "github.com/krackl1n/careeria/services/identity/internal/repository/password"
	phonechallengerepository "github.com/krackl1n/careeria/services/identity/internal/repository/phoneverificationchallenge"
	refreshtokenrepository "github.com/krackl1n/careeria/services/identity/internal/repository/refreshtokenfamily"
	securityeventrepository "github.com/krackl1n/careeria/services/identity/internal/repository/securityevent"
	sessionrepository "github.com/krackl1n/careeria/services/identity/internal/repository/session"
	signingkeyrepository "github.com/krackl1n/careeria/services/identity/internal/repository/signingkey"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	jwtsigner "github.com/krackl1n/careeria/services/identity/internal/signer/jwt"
	transportgrpc "github.com/krackl1n/careeria/services/identity/internal/transport/grpc"
	authenticationtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/authentication"
	metadatatransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/metadata"
	oauthtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/oauth"
	oauthclientadmintransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/oauthclientadmin"
	registrationtransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/registration"
	signingkeyadmintransport "github.com/krackl1n/careeria/services/identity/internal/transport/grpc/signingkeyadmin"
	authenticationusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/authentication"
	metadatausecase "github.com/krackl1n/careeria/services/identity/internal/usecase/metadata"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	oauthclientadminusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauthclientadmin"
	signingkeyadminusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/signingkeyadmin"
	"github.com/krackl1n/careeria/txmanager"
)

type App struct {
	dependencies       *Dependencies
	transactionManager *txmanager.Manager[pgx.Tx]
	clock              sharedclock.Clock
	idGenerator        *sharedidgen.UUID
	metrics            *sharedmetrics.Component
	grpcMetrics        *grpcinterceptor.Metrics
	grpcServer         *grpcserver.Server
	errorResolver      *transportgrpc.ErrorResolver

	authorizationTransactions *authorizationtransactionrepository.RedisRepository
	authorizationCodes        *authorizationcoderepository.RedisRepository
	authorizationGrants       *authorizationgrantrepository.PostgresRepository
	identities                *identityrepository.PostgresRepository
	identityProfiles          *identityprofilerepository.PostgresRepository
	passwords                 *passwordrepository.PostgresRepository
	oauthClients              *oauthclientrepository.PostgresRepository
	emailChallenges           *emailchallengerepository.PostgresRepository
	phoneChallenges           *phonechallengerepository.PostgresRepository
	sessions                  *sessionrepository.PostgresRepository
	refreshTokens             *refreshtokenrepository.PostgresRepository
	securityEvents            *securityeventrepository.PostgresRepository
	signingKeys               *signingkeyrepository.PostgresRepository
	outbox                    *outboxrepository.PostgresRepository
	kafkaProducer             *sharedkafka.Producer
	eventPublisher            *kafkaintegration.Publisher
	outboxWorker              *sharedoutbox.Worker[eventdomain.Message]
	authenticationFacts       *authenticationrepository.PostgresReader

	signer              *jwtsigner.Signer
	accessTokens        *accesstokenservice.Service
	idTokens            *idtokenservice.Service
	contactVerification *contactverificationservice.Service
	authentication      *authenticationusecase.Service
	oauth               *oauthusecase.Service
	metadata            *metadatausecase.Service
	oauthClientAdmin    *oauthclientadminusecase.Service
	signingKeyAdmin     *signingkeyadminusecase.Service

	authenticationHandler   *authenticationtransport.AuthenticationHandler
	registrationHandler     *registrationtransport.Handler
	oauthHandler            *oauthtransport.Handler
	metadataHandler         *metadatatransport.Handler
	oauthClientAdminHandler *oauthclientadmintransport.Handler
	signingKeyAdminHandler  *signingkeyadmintransport.Handler
}

func New(dependencies *Dependencies) (*App, error) {
	if dependencies == nil {
		return nil, errors.New("application dependencies are required")
	}
	if dependencies.Logger == nil {
		return nil, errors.New("application logger is required")
	}
	if dependencies.Closer == nil {
		return nil, errors.New("application closer is required")
	}
	if dependencies.PostgresClient == nil {
		return nil, errors.New("application postgres client is required")
	}
	if dependencies.RedisClient == nil {
		return nil, errors.New("application Redis client is required")
	}
	return &App{dependencies: dependencies}, nil
}

func (a *App) Runtimes(ctx context.Context) ([]sharedruntime.Component, error) {
	metrics, err := a.Metrics()
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}
	server, err := a.GRPCServer()
	if err != nil {
		return nil, fmt.Errorf("initialize gRPC server: %w", err)
	}
	outbox, err := a.OutboxComponent(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize outbox worker: %w", err)
	}
	if metrics.Enabled() {
		return []sharedruntime.Component{metrics, outbox, server}, nil
	}
	return []sharedruntime.Component{outbox, server}, nil
}
