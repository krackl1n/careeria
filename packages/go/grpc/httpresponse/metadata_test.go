package httpresponse

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestSetRedirect(t *testing.T) {
	stream := &serverTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)

	err := SetRedirect(ctx, http.StatusFound, "https://identity.example/login")

	require.NoError(t, err)
	require.Equal(t, []string{"302"}, stream.headers.Get(StatusCodeMetadataKey))
	require.Equal(t, []string{"https://identity.example/login"}, stream.headers.Get(LocationMetadataKey))
}

func TestSetRedirectRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		location   string
	}{
		{name: "non-redirect status", statusCode: http.StatusOK, location: "https://identity.example/login"},
		{name: "empty location", statusCode: http.StatusFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := SetRedirect(context.Background(), test.statusCode, test.location)
			require.Error(t, err)
		})
	}
}

type serverTransportStream struct {
	headers metadata.MD
}

func (*serverTransportStream) Method() string {
	return "/test.Service/Method"
}

func (s *serverTransportStream) SetHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (s *serverTransportStream) SendHeader(md metadata.MD) error {
	s.headers = metadata.Join(s.headers, md)
	return nil
}

func (*serverTransportStream) SetTrailer(metadata.MD) error {
	return nil
}
