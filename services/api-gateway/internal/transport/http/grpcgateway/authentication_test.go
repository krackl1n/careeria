package grpcgateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/http/handler"
	"github.com/stretchr/testify/require"
)

type authenticatorStub struct {
	principal ctxmeta.Principal
	err       error
	calls     int
}

func (a *authenticatorStub) Authenticate(*http.Request) (ctxmeta.Principal, error) {
	a.calls++
	return a.principal, a.err
}

func TestOptionalAuthenticationSkipsRequestWithoutBearerToken(t *testing.T) {
	authenticator := &authenticatorStub{err: errors.New("must not be called")}
	called := false

	middleware := optionalAuthentication(authenticator, handler.NewErrorHandler(nil))
	next := middleware(func(writer http.ResponseWriter, request *http.Request, _ map[string]string) {
		called = true
		_, ok := ctxmeta.PrincipalFromContext(request.Context())
		require.False(t, ok)
		writer.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	next(recorder, httptest.NewRequest(http.MethodPost, "/oauth2/token", nil), nil)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, called)
	require.Zero(t, authenticator.calls)
}

func TestOptionalAuthenticationAuthenticatesBearerAndPreservesTenantHeader(t *testing.T) {
	authenticator := &authenticatorStub{principal: ctxmeta.Principal{SubjectID: "user-1"}}
	called := false

	middleware := optionalAuthentication(authenticator, handler.NewErrorHandler(nil))
	next := middleware(func(writer http.ResponseWriter, request *http.Request, _ map[string]string) {
		called = true
		principal, ok := ctxmeta.PrincipalFromContext(request.Context())
		require.True(t, ok)
		require.Equal(t, "user-1", principal.SubjectID)
		require.Equal(t, "tenant-1", request.Header.Get("X-Tenant-ID"))
		writer.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("X-Tenant-ID", "tenant-1")
	recorder := httptest.NewRecorder()
	next(recorder, request, nil)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, called)
	require.Equal(t, 1, authenticator.calls)
}

func TestOptionalAuthenticationRejectsInvalidBearerToken(t *testing.T) {
	authenticator := &authenticatorStub{err: errors.New("invalid token")}
	called := false

	middleware := optionalAuthentication(authenticator, handler.NewErrorHandler(nil))
	next := middleware(func(http.ResponseWriter, *http.Request, map[string]string) {
		called = true
	})

	request := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	recorder := httptest.NewRecorder()
	next(recorder, request, nil)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":{"code":"unauthorized","message":"Unauthorized"}}`, recorder.Body.String())
	require.False(t, called)
	require.Equal(t, 1, authenticator.calls)
}

func TestOptionalAuthenticationDoesNotTreatBasicCredentialsAsBearer(t *testing.T) {
	authenticator := &authenticatorStub{err: errors.New("must not be called")}
	called := false
	middleware := optionalAuthentication(authenticator, handler.NewErrorHandler(nil))
	next := middleware(func(writer http.ResponseWriter, _ *http.Request, _ map[string]string) {
		called = true
		writer.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/oauth2/token", nil)
	request.Header.Set("Authorization", "Basic credentials")
	recorder := httptest.NewRecorder()
	next(recorder, request, nil)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, called)
	require.Zero(t, authenticator.calls)
}
