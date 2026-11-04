package middleware

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/stretchr/testify/require"
)

type authFunc func(*http.Request) (ctxmeta.Principal, error)

func (fn authFunc) Authenticate(r *http.Request) (ctxmeta.Principal, error) {
	return fn(r)
}

type authorizerFunc func(*http.Request) error

func (fn authorizerFunc) Authorize(r *http.Request) error {
	return fn(r)
}

type internalAuthFunc func(*http.Request) error

func (fn internalAuthFunc) AuthenticateInternal(r *http.Request) error {
	return fn(r)
}

type tenantProviderFunc func(*http.Request) (string, error)

func (fn tenantProviderFunc) ResolveTenant(r *http.Request) (string, error) {
	return fn(r)
}

type webhookVerifierFunc func(*http.Request, []byte) error

func (fn webhookVerifierFunc) Verify(r *http.Request, body []byte) error {
	return fn(r, body)
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		requestID       string
		correlationID   string
		wantSameID      bool
		wantCorrelation string
	}{
		{
			name:            "keeps valid request id",
			requestID:       "req-12345678",
			wantSameID:      true,
			wantCorrelation: "req-12345678",
		},
		{
			name:            "generates request id for invalid value",
			requestID:       "bad",
			wantSameID:      false,
			wantCorrelation: "",
		},
		{
			name:            "preserves existing correlation id",
			requestID:       "req-abcdefgh",
			correlationID:   "corr-1",
			wantSameID:      true,
			wantCorrelation: "corr-1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenRequestID string
			handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenRequestID = RequestIDFromRequest(r)
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestID != "" {
				req.Header.Set(RequestIDHeader, tt.requestID)
			}
			if tt.correlationID != "" {
				req.Header.Set(CorrelationIDHeader, tt.correlationID)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			require.NotEmpty(t, seenRequestID)
			require.Equal(t, seenRequestID, recorder.Header().Get(RequestIDHeader))
			require.Equal(t, seenRequestID, req.Header.Get(RequestIDHeader))
			if tt.wantSameID {
				require.Equal(t, tt.requestID, seenRequestID)
			}

			if tt.wantCorrelation != "" {
				require.Equal(t, tt.wantCorrelation, req.Header.Get(CorrelationIDHeader))
			} else {
				require.Equal(t, seenRequestID, req.Header.Get(CorrelationIDHeader))
			}
		})
	}
}

func TestHeaderNormalization(t *testing.T) {
	t.Parallel()

	mw := HeaderNormalization(HeaderNormalizationConfig{
		RemoveRequestHeaders: []string{"X-Remove"},
		SetRequestHeaders: map[string]string{
			"X-Set": "value",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Remove", "gone")
	recorder := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("X-Remove"))
		require.Equal(t, "value", r.Header.Get("X-Set"))
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	recorder := httptest.NewRecorder()

	SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
		HSTSMaxAge:            3600,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)

	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", recorder.Header().Get("X-Frame-Options"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	require.Equal(t, "default-src 'self'", recorder.Header().Get("Content-Security-Policy"))
	require.Equal(t, "max-age=3600; includeSubDomains; preload", recorder.Header().Get("Strict-Transport-Security"))
}

func TestRealIP(t *testing.T) {
	t.Parallel()

	trusted := netip.MustParsePrefix("10.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.2")

	recorder := httptest.NewRecorder()

	RealIP(RealIPConfig{TrustedProxies: []netip.Prefix{trusted}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "203.0.113.10", ClientIPFromRequest(r))
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthLikeMiddlewares(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantBody   string
	}{
		{
			name: "auth success",
			handler: Auth(authFunc(func(*http.Request) (ctxmeta.Principal, error) {
				return ctxmeta.Principal{SubjectID: "42"}, nil
			}), nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				principal, _ := ctxmeta.PrincipalFromContext(r.Context())
				_, _ = w.Write([]byte(principal.SubjectID))
			})),
			wantStatus: http.StatusOK,
			wantBody:   "42",
		},
		{
			name: "authorization failure",
			handler: Authorization(authorizerFunc(func(r *http.Request) error {
				return errors.New("deny")
			}), nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler must not be called")
			})),
			wantStatus: http.StatusForbidden,
			wantBody:   `"code":"forbidden"`,
		},
		{
			name: "internal auth failure",
			handler: InternalAuth(internalAuthFunc(func(r *http.Request) error {
				return errors.New("deny")
			}), nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler must not be called")
			})),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"code":"internal_unauthorized"`,
		},
		{
			name: "tenant resolver success",
			handler: TenantResolver(tenantProviderFunc(func(r *http.Request) (string, error) {
				return "tenant-1", nil
			}), nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(TenantIDFromRequest(r)))
			})),
			wantStatus: http.StatusOK,
			wantBody:   "tenant-1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.handler.ServeHTTP(recorder, req)
			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.wantBody)
		})
	}
}

func TestBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("uses default limit for non-positive limit", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(strings.Repeat("a", int(DefaultBodyLimit)+1)),
		)
		req.ContentLength = DefaultBodyLimit + 1
		recorder := httptest.NewRecorder()

		BodyLimit(0, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"code":"payload_too_large"`)
	})

	t.Run("panics on nil next handler", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(
			t,
			"body limit middleware: nil handler",
			func() {
				BodyLimit(5, nil)(nil)
			},
		)
	})

	t.Run("rejects large content length", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abcdef"))
		req.ContentLength = 6
		recorder := httptest.NewRecorder()

		BodyLimit(5, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"code":"payload_too_large"`)
	})

	t.Run("allows content length within limit and wraps body", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abc"))
		recorder := httptest.NewRecorder()
		called := false

		BodyLimit(5, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, "abc", string(body))
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(recorder, req)

		require.True(t, called)
		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("max bytes reader enforces limit for unknown content length", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abcdef"))
		req.ContentLength = -1
		recorder := httptest.NewRecorder()
		called := false

		BodyLimit(3, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			_, err := io.ReadAll(r.Body)
			var maxErr *http.MaxBytesError
			require.ErrorAs(t, err, &maxErr)
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(recorder, req)

		require.True(t, called)
		require.Equal(t, http.StatusNoContent, recorder.Code)
	})
}

func TestWebhookSignature(t *testing.T) {
	t.Parallel()

	verifiedBody := ""
	mw := WebhookSignature(WebhookSignatureConfig{
		Verifier: webhookVerifierFunc(func(r *http.Request, body []byte) error {
			verifiedBody = string(body)
			if string(body) == "bad" {
				return errors.New("invalid")
			}
			return nil
		}),
		MaxBodyBytes: 8,
	}, nil)

	t.Run("passes body through after verification", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("payload"))
		recorder := httptest.NewRecorder()

		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, "payload", string(body))
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(recorder, req)

		require.Equal(t, "payload", verifiedBody)
		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("rejects invalid signature", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("bad"))
		recorder := httptest.NewRecorder()

		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})).ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"code":"invalid_webhook_signature"`)
	})
}

func TestRecoverAndStructuredLogger(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(RequestIDHeader, "req-12345678")
	req.RemoteAddr = "203.0.113.10:1234"

	handler := RequestID()(StructuredLogger(logger)(Recover(logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"internal_server_error"`)
	require.Contains(t, buffer.String(), "http panic recovered")
	require.Contains(t, buffer.String(), "http request completed")
	require.Contains(t, buffer.String(), "req-12345678")
}

func TestReadBodyHelpers(t *testing.T) {
	t.Parallel()

	t.Run("read body limited returns max bytes error", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("toolong"))
		body, err := readBodyLimited(req, 3)
		require.Nil(t, body)
		var maxErr *http.MaxBytesError
		require.ErrorAs(t, err, &maxErr)
	})

	t.Run("read and restore body restores request body", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("payload"))
		body, err := readAndRestoreBody(req, 8)
		require.NoError(t, err)
		require.Equal(t, []byte("payload"), body)

		restored, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("payload"), restored)
	})
}
