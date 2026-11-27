package grpc

import (
	"errors"
	"fmt"
	"testing"

	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	"google.golang.org/grpc/codes"
)

func TestAuthorizationErrorMapping(t *testing.T) {
	tests := []struct {
		err  error
		code codes.Code
	}{
		{domain.ErrUnauthenticated, codes.Unauthenticated},
		{domain.ErrPermissionDenied, codes.PermissionDenied},
		{fmt.Errorf("%w: %w", domain.ErrAuthorizationUnavailable, errors.New("internal OpenFGA error")), codes.Unavailable},
	}
	for _, test := range tests {
		result, ok := (ErrorResolver{}).Resolve(test.err)
		if !ok || result.Code != test.code {
			t.Fatalf("resolve %v: %+v, %t", test.err, result, ok)
		}
		if test.code == codes.Unavailable && result.Message != "authorization service unavailable" {
			t.Fatalf("authorization error leaked: %s", result.Message)
		}
	}
}
