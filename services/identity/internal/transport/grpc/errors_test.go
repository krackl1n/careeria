package grpc

import (
	"errors"
	"testing"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func TestErrorResolver(t *testing.T) {
	resolved, ok := NewErrorResolver().Resolve(apperr.New(
		apperr.TypeInvalidArgument,
		"oauth_invalid_grant",
		"OAuth grant is invalid",
		errors.New("authorization code expired"),
	))

	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, resolved.Code)
	require.Equal(t, "OAuth grant is invalid", resolved.Message)
	require.Len(t, resolved.Details, 1)
	info, ok := resolved.Details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "oauth_invalid_grant", info.GetReason())
	require.Equal(t, "careeria.identity", info.GetDomain())
}

func TestErrorResolverIgnoresUnknownErrors(t *testing.T) {
	_, ok := NewErrorResolver().Resolve(errors.New("database failed"))

	require.False(t, ok)
}
