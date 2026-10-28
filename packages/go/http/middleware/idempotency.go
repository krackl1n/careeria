package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

const IdempotencyKeyHeader = "Idempotency-Key"

type IdempotencyState string

const (
	IdempotencyStarted  IdempotencyState = "started"
	IdempotencyReplay   IdempotencyState = "replay"
	IdempotencyLocked   IdempotencyState = "locked"
	IdempotencyConflict IdempotencyState = "conflict"
)

type IdempotencyStore interface {
	Start(ctx context.Context, key string, fingerprint string, ttl time.Duration) (IdempotencyStartResult, error)
	Finish(ctx context.Context, key string, fingerprint string, record IdempotencyRecord, ttl time.Duration) error
	Fail(ctx context.Context, key string, fingerprint string) error
}

type IdempotencyStartResult struct {
	State  IdempotencyState
	Record IdempotencyRecord
}

type IdempotencyRecord struct {
	Status int
	Header http.Header
	Body   []byte
}

type IdempotencyConfig struct {
	Store IdempotencyStore

	Header string

	LockTTL   time.Duration
	ResultTTL time.Duration

	MaxRequestBodyBytes  int64
	MaxResponseBodyBytes int64

	Fingerprint func(r *http.Request, body []byte) (string, error)
	StoreStatus func(status int) bool
}

func Idempotency(config IdempotencyConfig, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if config.Store == nil {
		panic("idempotency middleware: nil store")
	}

	if config.Header == "" {
		config.Header = IdempotencyKeyHeader
	}

	if config.LockTTL <= 0 {
		config.LockTTL = time.Minute
	}

	if config.ResultTTL <= 0 {
		config.ResultTTL = 24 * time.Hour
	}

	if config.MaxRequestBodyBytes <= 0 {
		config.MaxRequestBodyBytes = DefaultBodyLimit
	}

	if config.MaxResponseBodyBytes <= 0 {
		config.MaxResponseBodyBytes = DefaultBodyLimit
	}

	if config.Fingerprint == nil {
		config.Fingerprint = defaultIdempotencyFingerprint
	}

	if config.StoreStatus == nil {
		config.StoreStatus = func(status int) bool {
			return status >= http.StatusOK && status < http.StatusInternalServerError
		}
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("idempotency middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(config.Header))
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := readAndRestoreBody(r, config.MaxRequestBodyBytes)
			if err != nil {
				errorHandler.Handle(w, r, err)
				return
			}

			fingerprint, err := config.Fingerprint(r, body)
			if err != nil {
				errorHandler.Handle(w, r, httperror.Wrap(
					http.StatusConflict,
					"idempotency_fingerprint_failed",
					"Idempotency fingerprint failed",
					err,
				))
				return
			}

			result, err := config.Store.Start(r.Context(), key, fingerprint, config.LockTTL)
			if err != nil {
				errorHandler.Handle(w, r, httperror.Wrap(
					http.StatusConflict,
					"idempotency_conflict",
					"Idempotency conflict",
					err,
				))
				return
			}

			switch result.State {
			case IdempotencyReplay:
				for name, values := range result.Record.Header {
					for _, value := range values {
						w.Header().Add(name, value)
					}
				}

				w.Header().Set("Idempotent-Replayed", "true")
				w.WriteHeader(result.Record.Status)

				_, _ = w.Write(result.Record.Body)
				return

			case IdempotencyLocked:
				errorHandler.Handle(w, r, httperror.New(
					http.StatusConflict,
					"idempotency_in_progress",
					"Request with this idempotency key is already in progress",
				))
				return

			case IdempotencyConflict:
				errorHandler.Handle(w, r, httperror.New(
					http.StatusConflict,
					"idempotency_key_conflict",
					"Idempotency key was already used with different request",
				))
				return

			case IdempotencyStarted:
			default:
				errorHandler.Handle(w, r, httperror.New(
					http.StatusConflict,
					"idempotency_conflict",
					"Idempotency conflict",
				))
				return
			}

			released := false

			defer func() {
				if !released {
					_ = config.Store.Fail(r.Context(), key, fingerprint)
				}
			}()

			writer := &idempotencyWriter{
				ResponseWriter: w,
				maxBodyBytes:   config.MaxResponseBodyBytes,
			}

			next.ServeHTTP(writer, r)

			status := writer.statusCode()

			if !config.StoreStatus(status) || writer.overflow {
				released = true
				_ = config.Store.Fail(r.Context(), key, fingerprint)
				return
			}

			released = true

			_ = config.Store.Finish(r.Context(), key, fingerprint, IdempotencyRecord{
				Status: status,
				Header: writer.header.Clone(),
				Body:   writer.body.Bytes(),
			}, config.ResultTTL)
		})
	}
}

func readAndRestoreBody(r *http.Request, limit int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > limit {
		return nil, &http.MaxBytesError{Limit: limit}
	}

	if err := r.Body.Close(); err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

func defaultIdempotencyFingerprint(r *http.Request, body []byte) (string, error) {
	hash := sha256.Sum256(body)

	parts := []string{
		r.Method,
		r.URL.Path,
		TenantIDFromRequest(r),
		r.Header.Get("X-User-ID"),
		hex.EncodeToString(hash[:]),
	}

	return strings.Join(parts, ":"), nil
}

type idempotencyWriter struct {
	http.ResponseWriter

	status int
	wrote  bool

	header http.Header

	body         bytes.Buffer
	maxBodyBytes int64
	overflow     bool
}

func (w *idempotencyWriter) Header() http.Header {
	if w.header == nil {
		w.header = w.ResponseWriter.Header().Clone()
	}

	return w.ResponseWriter.Header()
}

func (w *idempotencyWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}

	w.status = status
	w.wrote = true
	w.header = w.ResponseWriter.Header().Clone()
	w.ResponseWriter.WriteHeader(status)
}

func (w *idempotencyWriter) Write(data []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}

	if !w.overflow {
		if int64(w.body.Len()+len(data)) <= w.maxBodyBytes {
			_, _ = w.body.Write(data)
		} else {
			w.overflow = true
		}
	}

	return w.ResponseWriter.Write(data)
}

func (w *idempotencyWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *idempotencyWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}

	return w.status
}
