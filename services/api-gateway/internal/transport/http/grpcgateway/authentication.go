package grpcgateway

import (
	"net/http"
	"strings"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
	httpmiddleware "github.com/krackl1n/careeria/http/middleware"
)

func optionalAuthentication(
	authenticator httpmiddleware.Authenticator,
	errorHandler handler.ErrorHandler,
) gatewayruntime.Middleware {
	if authenticator == nil {
		panic("gRPC gateway authentication middleware: nil authenticator")
	}
	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next gatewayruntime.HandlerFunc) gatewayruntime.HandlerFunc {
		if next == nil {
			panic("gRPC gateway authentication middleware: nil handler")
		}

		return func(writer http.ResponseWriter, request *http.Request, pathParams map[string]string) {
			authorization := strings.Fields(request.Header.Get("Authorization"))
			if len(authorization) == 0 || !strings.EqualFold(authorization[0], "Bearer") {
				next(writer, request, pathParams)
				return
			}

			principal, err := authenticator.Authenticate(request)
			if err != nil || principal.SubjectID == "" {
				errorHandler.Handle(writer, request, httperror.New(
					http.StatusUnauthorized,
					"unauthorized",
					http.StatusText(http.StatusUnauthorized),
				))
				return
			}

			ctx := ctxmeta.WithPrincipal(request.Context(), principal)
			next(writer, request.WithContext(ctx), pathParams)
		}
	}
}
