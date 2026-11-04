package httptransport

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	httppreset "github.com/krackl1n/careeria/http/middleware/preset"
	swaggertransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http/swagger"
)

type RouterDeps struct {
	SwaggerConfig swaggertransport.Config
	GatewayConfig httppreset.GatewayConfig
	GRPCGateway   http.Handler

	SwaggerSpecification []byte

	MiddlewarePresetDeps httppreset.GatewayDeps
}

func NewRouter(deps RouterDeps) (*chi.Mux, error) {
	if deps.GRPCGateway == nil {
		return nil, errors.New("gRPC gateway mux is required")
	}

	router := chi.NewRouter()
	router.Use(httppreset.Gateway(deps.GatewayConfig, deps.MiddlewarePresetDeps)...)

	if deps.SwaggerConfig.Enabled {
		docs, err := swaggertransport.New(deps.SwaggerConfig, deps.SwaggerSpecification)
		if err != nil {
			return nil, err
		}

		router.HandleFunc(deps.SwaggerConfig.SpecificationPath, docs.OpenAPI)
		router.Handle(deps.SwaggerConfig.Path+"*", docs.UI())
	}

	router.Mount("/", deps.GRPCGateway)

	return router, nil
}
