package grpcgateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	"github.com/krackl1n/careeria/ctxmeta"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"github.com/krackl1n/careeria/http/handler"
	httpmiddleware "github.com/krackl1n/careeria/http/middleware"
	identityintegration "github.com/krackl1n/careeria/services/api-gateway/internal/integrations/identity"
	"google.golang.org/grpc/metadata"
)

type MuxDeps struct {
	Logger        *slog.Logger
	Authenticator httpmiddleware.Authenticator
	Identity      *identityintegration.Clients
}

func NewMux(ctx context.Context, deps MuxDeps) (*gatewayruntime.ServeMux, error) {
	if ctx == nil {
		return nil, errors.New("gateway context is required")
	}
	if deps.Identity == nil {
		return nil, errors.New("identity integration is required")
	}
	if deps.Authenticator == nil {
		return nil, errors.New("gateway authenticator is required")
	}

	errorHandler := handler.NewErrorHandler(deps.Logger, ResolveError)
	mux := gatewayruntime.NewServeMux(
		gatewayruntime.WithMiddlewares(optionalAuthentication(deps.Authenticator, errorHandler)),
		gatewayruntime.WithMetadata(principalMetadata),
		gatewayruntime.WithMarshalerOption(
			"application/x-www-form-urlencoded",
			&FormMarshaler{},
		),
		gatewayruntime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
		gatewayruntime.WithOutgoingHeaderMatcher(outgoingHeaderMatcher),
		gatewayruntime.WithForwardResponseOption(ApplyHTTPResponseMetadata),
		gatewayruntime.WithErrorHandler(ErrorHandler(errorHandler)),
	)

	if err := registerIdentity(ctx, mux, deps.Identity); err != nil {
		return nil, err
	}

	return mux, nil
}

func registerIdentity(ctx context.Context, mux *gatewayruntime.ServeMux, clients *identityintegration.Clients) error {
	if err := identityv1.RegisterOAuthServiceHandlerClient(ctx, mux, clients.OAuth); err != nil {
		return fmt.Errorf("register OAuth gateway handlers: %w", err)
	}
	if err := identityv1.RegisterAuthenticationServiceHandlerClient(ctx, mux, clients.Authentication); err != nil {
		return fmt.Errorf("register authentication gateway handlers: %w", err)
	}
	if err := identityv1.RegisterIdentitySessionServiceHandlerClient(ctx, mux, clients.Session); err != nil {
		return fmt.Errorf("register identity session gateway handlers: %w", err)
	}
	if err := identityv1.RegisterIdentityMetadataServiceHandlerClient(ctx, mux, clients.Metadata); err != nil {
		return fmt.Errorf("register identity metadata gateway handlers: %w", err)
	}
	if err := identityv1.RegisterRegistrationServiceHandlerClient(ctx, mux, clients.Registration); err != nil {
		return fmt.Errorf("register registration gateway handlers: %w", err)
	}
	return nil
}

func incomingHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "authorization", "cookie", "x-tenant-id":
		return strings.ToLower(key), true
	default:
		return gatewayruntime.DefaultHeaderMatcher(key)
	}
}

func outgoingHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "set-cookie":
		return "Set-Cookie", true
	case "www-authenticate":
		return "WWW-Authenticate", true
	case grpchttpresponse.StatusCodeMetadataKey, grpchttpresponse.LocationMetadataKey:
		return "", false
	}

	return "Grpc-Metadata-" + key, true
}

func principalMetadata(ctx context.Context, _ *http.Request) metadata.MD {
	principal, ok := ctxmeta.PrincipalFromContext(ctx)
	if !ok {
		return nil
	}

	return metadata.Pairs("x-principal-sub", principal.SubjectID)
}
