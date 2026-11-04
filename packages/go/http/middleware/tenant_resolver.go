package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

const TenantIDHeader = "X-Tenant-ID"

type TenantProvider interface {
	ResolveTenant(r *http.Request) (string, error)
}

func TenantResolver(provider TenantProvider, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if provider == nil {
		panic("tenant resolver middleware: nil provider")
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("tenant resolver middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, err := provider.ResolveTenant(r)
			if err != nil || tenantID == "" {
				errorHandler.Handle(w, r, httperror.New(
					http.StatusBadRequest,
					"tenant_required",
					"Tenant is required",
				))
				return
			}

			r.Header.Set(TenantIDHeader, tenantID)

			ctx := ctxmeta.WithTenantID(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TenantIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if value, ok := ctxmeta.TenantID(r.Context()); ok {
		return value
	}

	return r.Header.Get(TenantIDHeader)
}
