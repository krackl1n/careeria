package grpcgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/krackl1n/careeria/http/handler"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorHandler(
	httpErrorHandler handler.ErrorHandler,
) gatewayruntime.ErrorHandlerFunc {
	if httpErrorHandler == nil {
		httpErrorHandler = handler.NewErrorHandler(nil, ResolveError)
	}

	return func(
		_ context.Context,
		_ *gatewayruntime.ServeMux,
		_ gatewayruntime.Marshaler,
		writer http.ResponseWriter,
		request *http.Request,
		err error,
	) {
		if request.URL.Path == "/oauth2/token" {
			writeOAuthError(writer, err)
			return
		}
		httpErrorHandler.Handle(writer, request, err)
	}
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func writeOAuthError(writer http.ResponseWriter, err error) {
	response, statusCode := resolveOAuthError(err)
	writer.Header().Set("Content-Type", "application/json")
	if response.Error == "invalid_client" {
		writer.Header().Set("WWW-Authenticate", `Basic realm="oauth2/token"`)
	}
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(response)
}

func resolveOAuthError(err error) (oauthErrorResponse, int) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return oauthErrorResponse{
			Error:            "server_error",
			ErrorDescription: "OAuth token request failed",
		}, http.StatusInternalServerError
	}

	for _, detail := range grpcStatus.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.GetDomain() != "careeria.identity" {
			continue
		}
		errorCode := strings.TrimPrefix(info.GetReason(), "oauth_")
		return oauthErrorResponse{
			Error:            errorCode,
			ErrorDescription: grpcStatus.Message(),
		}, oauthHTTPStatus(errorCode)
	}

	switch grpcStatus.Code() {
	case codes.InvalidArgument:
		return oauthErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "OAuth token request is invalid",
		}, http.StatusBadRequest
	case codes.Unauthenticated:
		return oauthErrorResponse{
			Error:            "invalid_client",
			ErrorDescription: "OAuth client authentication failed",
		}, http.StatusUnauthorized
	case codes.Unavailable:
		return oauthErrorResponse{
			Error:            "temporarily_unavailable",
			ErrorDescription: "OAuth token endpoint is temporarily unavailable",
		}, http.StatusServiceUnavailable
	default:
		return oauthErrorResponse{
			Error:            "server_error",
			ErrorDescription: "OAuth token request failed",
		}, http.StatusInternalServerError
	}
}

func oauthHTTPStatus(errorCode string) int {
	switch errorCode {
	case "invalid_client":
		return http.StatusUnauthorized
	case "temporarily_unavailable":
		return http.StatusServiceUnavailable
	case "server_error":
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func ResolveError(err error) (handler.ResolvedError, bool) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return handler.ResolvedError{}, false
	}

	resolved := handler.ResolvedError{
		Status:  gatewayruntime.HTTPStatusFromCode(grpcStatus.Code()),
		Code:    codeName(grpcStatus.Code()),
		Message: grpcStatus.Message(),
	}

	if details, ok := validationDetails(grpcStatus); ok {
		resolved.Code = "validation_failed"
		resolved.Message = "request validation failed"
		resolved.Details = details
	} else if code, ok := applicationErrorCode(grpcStatus); ok {
		resolved.Code = code
	} else if grpcStatus.Code() == codes.InvalidArgument {
		resolved.Message = "request contains invalid arguments"
	}

	return resolved, true
}

func applicationErrorCode(grpcStatus *status.Status) (string, bool) {
	for _, detail := range grpcStatus.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || !strings.HasPrefix(info.GetDomain(), "careeria.") || info.GetReason() == "" {
			continue
		}
		return info.GetReason(), true
	}
	return "", false
}

type validationErrorDetails struct {
	Fields []fieldViolation `json:"fields"`
}

type fieldViolation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func validationDetails(
	grpcStatus *status.Status,
) (validationErrorDetails, bool) {
	if grpcStatus.Code() != codes.InvalidArgument {
		return validationErrorDetails{}, false
	}

	result := validationErrorDetails{}

	for _, detail := range grpcStatus.Details() {
		badRequest, ok := detail.(*errdetails.BadRequest)
		if !ok {
			continue
		}

		for _, violation := range badRequest.GetFieldViolations() {
			code := violation.GetReason()
			if code == "" {
				code = "invalid"
			}

			message := violation.GetDescription()
			if message == "" {
				message = "field is invalid"
			}

			result.Fields = append(result.Fields, fieldViolation{
				Field:   violation.GetField(),
				Code:    code,
				Message: message,
			})
		}
	}

	return result, len(result.Fields) > 0
}

func codeName(code codes.Code) string {
	switch code {
	case codes.Canceled:
		return "canceled"
	case codes.Unknown:
		return "unknown"
	case codes.InvalidArgument:
		return "invalid_argument"
	case codes.DeadlineExceeded:
		return "deadline_exceeded"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "already_exists"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "resource_exhausted"
	case codes.FailedPrecondition:
		return "failed_precondition"
	case codes.Aborted:
		return "aborted"
	case codes.OutOfRange:
		return "out_of_range"
	case codes.Unimplemented:
		return "unimplemented"
	case codes.Internal:
		return "internal"
	case codes.Unavailable:
		return "unavailable"
	case codes.DataLoss:
		return "data_loss"
	case codes.Unauthenticated:
		return "unauthenticated"
	default:
		return "internal_error"
	}
}
