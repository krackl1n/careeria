package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	swaggertransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http/swagger"
	"github.com/stretchr/testify/require"
)

func TestNewRouterMountsGatewayAndSwaggerHandlers(t *testing.T) {
	gateway := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})

	router, err := NewRouter(RouterDeps{
		GRPCGateway: gateway,
		SwaggerConfig: swaggertransport.Config{
			Enabled:           true,
			Title:             "API",
			Path:              "/swagger/",
			SpecificationPath: "/openapi/swagger.json",
		},
		SwaggerSpecification: []byte(`{"openapi":"3.0.0"}`),
	})
	require.NoError(t, err)

	tests := []struct {
		path   string
		status int
	}{
		{path: "/api/v1/meetings", status: http.StatusNoContent},
		{path: "/swagger/", status: http.StatusOK},
		{path: "/openapi/swagger.json", status: http.StatusOK},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		require.Equal(t, test.status, recorder.Code)
	}
}

func TestNewRouterRequiresGatewayMux(t *testing.T) {
	_, err := NewRouter(RouterDeps{})
	require.EqualError(t, err, "gRPC gateway mux is required")
}
