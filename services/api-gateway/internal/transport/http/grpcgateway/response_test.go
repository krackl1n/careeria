package grpcgateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestApplyHTTPResponseMetadataRedirect(t *testing.T) {
	ctx := responseMetadataContext(metadata.Pairs(
		grpchttpresponse.StatusCodeMetadataKey, "302",
		grpchttpresponse.LocationMetadataKey, "https://identity.example/login",
	))
	recorder := httptest.NewRecorder()

	err := ApplyHTTPResponseMetadata(ctx, recorder, nil)

	require.NoError(t, err)
	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "https://identity.example/login", recorder.Header().Get("Location"))
}

func TestApplyHTTPResponseMetadataIgnoresResponseWithoutStatus(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := ApplyHTTPResponseMetadata(context.Background(), recorder, nil)

	require.NoError(t, err)
	require.Empty(t, recorder.Header().Get("Location"))
}

func TestApplyHTTPResponseMetadataRejectsInvalidRedirect(t *testing.T) {
	tests := []struct {
		name     string
		metadata metadata.MD
	}{
		{
			name:     "invalid status",
			metadata: metadata.Pairs(grpchttpresponse.StatusCodeMetadataKey, "found"),
		},
		{
			name:     "missing location",
			metadata: metadata.Pairs(grpchttpresponse.StatusCodeMetadataKey, "302"),
		},
		{
			name: "multiple locations",
			metadata: metadata.Pairs(
				grpchttpresponse.StatusCodeMetadataKey, "302",
				grpchttpresponse.LocationMetadataKey, "https://identity.example/login",
				grpchttpresponse.LocationMetadataKey, "https://identity.example/other",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			err := ApplyHTTPResponseMetadata(responseMetadataContext(test.metadata), recorder, nil)

			require.Error(t, err)
			require.Empty(t, recorder.Header().Get("Location"))
		})
	}
}

func responseMetadataContext(md metadata.MD) context.Context {
	return gatewayruntime.NewServerMetadataContext(
		context.Background(),
		gatewayruntime.ServerMetadata{HeaderMD: md},
	)
}
