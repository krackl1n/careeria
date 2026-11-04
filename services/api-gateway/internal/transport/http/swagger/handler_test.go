package swagger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	contractsopenapi "github.com/krackl1n/careeria/contracts/openapi"
	"github.com/stretchr/testify/require"
)

var testConfig = Config{
	Enabled:           true,
	Title:             "RVSpace API",
	Path:              "/swagger/",
	SpecificationPath: "/openapi/careeria.swagger.json",
}

func TestHandlerOpenAPI(t *testing.T) {
	t.Parallel()

	handler, err := New(testConfig, contractsopenapi.Specification())
	require.NoError(t, err)
	recorder := httptest.NewRecorder()

	handler.OpenAPI(recorder, httptest.NewRequest(http.MethodGet, testConfig.SpecificationPath, nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, "no-store, max-age=0", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
	require.JSONEq(t, string(contractsopenapi.Specification()), recorder.Body.String())
}

func TestHandlerUI(t *testing.T) {
	t.Parallel()

	handler, err := New(testConfig, contractsopenapi.Specification())
	require.NoError(t, err)
	recorder := httptest.NewRecorder()

	handler.UI().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, testConfig.Path, nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	require.Contains(t, recorder.Body.String(), testConfig.SpecificationPath+"?v=")
}
