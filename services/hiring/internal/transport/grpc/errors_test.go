package grpc

import (
	"errors"
	"testing"

	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func TestErrorResolver(t *testing.T) {
	resolver := NewErrorResolver()

	t.Run("resolves domain apperr", func(t *testing.T) {
		err := apperr.NotFound("application_not_found", "application was not found", errors.New("db miss"))
		resolved, ok := resolver.Resolve(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, resolved.Code)
		require.Equal(t, "application was not found", resolved.Message)
		require.Len(t, resolved.Details, 1)

		info, ok := resolved.Details[0].(*errdetails.ErrorInfo)
		require.True(t, ok)
		require.Equal(t, "application_not_found", info.GetReason())
		require.Equal(t, "careeria.hiring", info.GetDomain())
	})

	t.Run("ignores unknown errors", func(t *testing.T) {
		err := errors.New("unknown error")
		_, ok := resolver.Resolve(err)
		require.False(t, ok)
	})
}
