package grpcgateway

import (
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"github.com/stretchr/testify/require"
)

func TestPrincipalMetadata(t *testing.T) {
	ctx := ctxmeta.WithPrincipal(t.Context(), ctxmeta.Principal{SubjectID: "user-1"})

	values := principalMetadata(ctx, nil)
	require.Equal(t, []string{"user-1"}, values.Get("x-principal-sub"))
	require.Empty(t, values.Get("x-tenant-id"))
}

func TestIncomingHeaderMatcher(t *testing.T) {
	key, ok := incomingHeaderMatcher("Authorization")
	require.True(t, ok)
	require.Equal(t, "authorization", key)

	key, ok = incomingHeaderMatcher("Cookie")
	require.True(t, ok)
	require.Equal(t, "cookie", key)
}

func TestOutgoingHeaderMatcherHidesHTTPResponseMetadata(t *testing.T) {
	for _, key := range []string{
		grpchttpresponse.StatusCodeMetadataKey,
		grpchttpresponse.LocationMetadataKey,
	} {
		mapped, ok := outgoingHeaderMatcher(key)
		require.False(t, ok)
		require.Empty(t, mapped)
	}
}
