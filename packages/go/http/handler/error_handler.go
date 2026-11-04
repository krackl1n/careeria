package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

const RequestIDHeader = "X-Request-ID"

type ErrorHandler interface {
	Handle(http.ResponseWriter, *http.Request, error)
}

type JSONErrorHandler struct {
	logger   *slog.Logger
	resolver Resolver
}

func NewErrorHandler(
	logger *slog.Logger,
	resolvers ...Resolver,
) *JSONErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}

	chain := make([]Resolver, 0, len(resolvers)+2)
	chain = append(chain, resolvers...)
	chain = append(chain, resolveHTTPError)
	chain = append(chain, DefaultResolver)

	return &JSONErrorHandler{
		logger:   logger,
		resolver: Chain(chain...),
	}
}

func (h *JSONErrorHandler) Handle(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if err == nil {
		return
	}

	resolved, ok := h.resolver(err)
	if !ok {
		resolved = internalError()
	}

	resolved = sanitize(resolved)

	if resolved.Status >= http.StatusInternalServerError {
		h.logger.ErrorContext(
			r.Context(),
			"http request failed",
			slog.Any("error", err),
			slog.Int("status", resolved.Status),
			slog.String("code", resolved.Code),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", requestID(r)),
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resolved.Status)

	encodeErr := json.NewEncoder(w).Encode(errorResponse{
		Error: errorBody{
			Code:    resolved.Code,
			Message: resolved.Message,
			Details: resolved.Details,
		},
		RequestID: requestID(r),
	})
	if encodeErr != nil {
		h.logger.ErrorContext(
			r.Context(),
			"encode HTTP error response",
			slog.Any("error", encodeErr),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", requestID(r)),
		)
	}
}

type errorResponse struct {
	Error     errorBody `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func sanitize(err ResolvedError) ResolvedError {
	if err.Status < 400 || err.Status > 599 {
		return internalError()
	}

	if err.Status >= http.StatusInternalServerError {
		err.Code = statusCode(err.Status)
		err.Message = http.StatusText(err.Status)
		err.Details = nil

		return err
	}

	if err.Code == "" {
		err.Code = statusCode(err.Status)
	}

	if err.Message == "" {
		err.Message = http.StatusText(err.Status)
	}

	return err
}

func internalError() ResolvedError {
	return ResolvedError{
		Status:  http.StatusInternalServerError,
		Code:    statusCode(http.StatusInternalServerError),
		Message: http.StatusText(http.StatusInternalServerError),
	}
}

func statusCode(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "http_error"
	}

	return strings.ToLower(strings.ReplaceAll(text, " ", "_"))
}

func requestID(r *http.Request) string {
	if r == nil {
		return ""
	}

	return r.Header.Get(RequestIDHeader)
}
