package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/krackl1n/careeria/http/httperror"
)

const statusClientClosedRequest = 499

type ResolvedError struct {
	Status  int
	Code    string
	Message string
	Details any
}

type Resolver func(error) (ResolvedError, bool)

func Chain(resolvers ...Resolver) Resolver {
	return func(err error) (ResolvedError, bool) {
		for _, resolver := range resolvers {
			if resolver == nil {
				continue
			}

			resolved, ok := resolver(err)
			if ok {
				return resolved, true
			}
		}

		return ResolvedError{}, false
	}
}

func DefaultResolver(err error) (ResolvedError, bool) {
	if err == nil {
		return ResolvedError{}, false
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return ResolvedError{
			Status:  http.StatusRequestEntityTooLarge,
			Code:    "payload_too_large",
			Message: "request body is too large",
		}, true
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return ResolvedError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_json",
			Message: "request body contains invalid JSON",
		}, true
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return ResolvedError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_json_type",
			Message: "request body contains an invalid value type",
		}, true
	}

	switch {
	case errors.Is(err, io.EOF):
		return ResolvedError{
			Status:  http.StatusBadRequest,
			Code:    "empty_request_body",
			Message: "request body is required",
		}, true

	case errors.Is(err, context.DeadlineExceeded):
		return ResolvedError{
			Status:  http.StatusGatewayTimeout,
			Code:    "gateway_timeout",
			Message: http.StatusText(http.StatusGatewayTimeout),
		}, true

	case errors.Is(err, context.Canceled):
		return ResolvedError{
			Status:  statusClientClosedRequest,
			Code:    "client_closed_request",
			Message: "Client Closed Request",
		}, true

	default:
		return ResolvedError{}, false
	}
}

func resolveHTTPError(err error) (ResolvedError, bool) {
	if err == nil {
		return ResolvedError{}, false
	}

	var httpErr httperror.Error
	if errors.As(err, &httpErr) {
		return ResolvedError{
			Status:  httpErr.Status,
			Code:    httpErr.Code,
			Message: httpErr.Message,
			Details: httpErr.Details,
		}, true
	}

	var httpErrPtr *httperror.Error
	if errors.As(err, &httpErrPtr) && httpErrPtr != nil {
		return ResolvedError{
			Status:  httpErrPtr.Status,
			Code:    httpErrPtr.Code,
			Message: httpErrPtr.Message,
			Details: httpErrPtr.Details,
		}, true
	}

	return ResolvedError{}, false
}
