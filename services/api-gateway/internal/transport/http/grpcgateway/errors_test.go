package grpcgateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/krackl1n/careeria/http/handler"
)

func TestResolveError(t *testing.T) {
	resolved, ok := ResolveError(status.Error(codes.NotFound, "meeting not found"))
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, resolved.Status)
	require.Equal(t, "not_found", resolved.Code)
	require.Equal(t, "meeting not found", resolved.Message)
}

func TestResolveErrorHidesInvalidArgumentInternals(t *testing.T) {
	resolved, ok := ResolveError(status.Error(
		codes.InvalidArgument,
		`proto: (line 2:22): invalid value for int64 field value: "string"`,
	))

	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, resolved.Status)
	require.Equal(t, "invalid_argument", resolved.Code)
	require.Equal(t, "request contains invalid arguments", resolved.Message)
}

func TestResolveErrorIncludesValidationDetails(t *testing.T) {
	grpcStatus, err := status.New(
		codes.InvalidArgument,
		"invalid request",
	).WithDetails(&errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "email",
				Reason:      "email",
				Description: "must be a valid email address",
			},
		},
	})
	require.NoError(t, err)

	resolved, ok := ResolveError(grpcStatus.Err())
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, resolved.Status)
	require.Equal(t, "validation_failed", resolved.Code)
	require.Equal(t, "request validation failed", resolved.Message)
	require.Equal(t, validationErrorDetails{
		Fields: []fieldViolation{
			{
				Field:   "email",
				Code:    "email",
				Message: "must be a valid email address",
			},
		},
	}, resolved.Details)
}

func TestResolveErrorPreservesApplicationInvalidArgument(t *testing.T) {
	grpcStatus, err := status.New(codes.InvalidArgument, `Ошибка в синтаксисе шаблона: unexpected token "|"`).WithDetails(&errdetails.ErrorInfo{
		Reason: "template_syntax_invalid",
		Domain: "careeria.document",
	})
	require.NoError(t, err)

	resolved, ok := ResolveError(grpcStatus.Err())
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, resolved.Status)
	require.Equal(t, "template_syntax_invalid", resolved.Code)
	require.Equal(t, `Ошибка в синтаксисе шаблона: unexpected token "|"`, resolved.Message)
}

func TestErrorHandlerUsesSharedJSONEnvelope(t *testing.T) {
	httpErrorHandler := handler.NewErrorHandler(
		slog.Default(),
		ResolveError,
	)
	callback := ErrorHandler(httpErrorHandler)

	request := httptest.NewRequest(http.MethodGet, "/v1/meetings/42", nil)
	request.Header.Set(handler.RequestIDHeader, "request-123")
	recorder := httptest.NewRecorder()

	validationErr := validationStatus(t)

	callback(
		context.Background(),
		nil,
		nil,
		recorder,
		request,
		validationErr,
	)

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details struct {
				Fields []fieldViolation `json:"fields"`
			} `json:"details"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "validation_failed", response.Error.Code)
	require.Equal(t, "request validation failed", response.Error.Message)
	require.Equal(t, []fieldViolation{
		{
			Field:   "email",
			Code:    "email",
			Message: "must be a valid email address",
		},
	}, response.Error.Details.Fields)
	require.Equal(t, "request-123", response.RequestID)
}

func TestErrorHandlerUsesOAuthErrorContractForTokenEndpoint(t *testing.T) {
	callback := ErrorHandler(nil)
	request := httptest.NewRequest(http.MethodPost, "/oauth2/token", nil)
	recorder := httptest.NewRecorder()
	grpcStatus, err := status.New(
		codes.Unauthenticated,
		"OAuth client authentication failed",
	).WithDetails(&errdetails.ErrorInfo{
		Reason: "oauth_invalid_client",
		Domain: "careeria.identity",
	})
	require.NoError(t, err)

	callback(
		context.Background(),
		nil,
		nil,
		recorder,
		request,
		grpcStatus.Err(),
	)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, `Basic realm="oauth2/token"`, recorder.Header().Get("WWW-Authenticate"))
	var response oauthErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "invalid_client", response.Error)
	require.Equal(t, "OAuth client authentication failed", response.ErrorDescription)
}

func validationStatus(t *testing.T) error {
	t.Helper()

	grpcStatus, err := status.New(
		codes.InvalidArgument,
		"invalid request",
	).WithDetails(&errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "email",
				Reason:      "email",
				Description: "must be a valid email address",
			},
		},
	})
	require.NoError(t, err)

	return grpcStatus.Err()
}
