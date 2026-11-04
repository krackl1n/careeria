package client

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestBuildTransportAppliesDefaults(t *testing.T) {
	t.Parallel()

	transport := buildTransport(Config{})

	require.Equal(t, 100, transport.MaxIdleConns)
	require.Equal(t, 20, transport.MaxIdleConnsPerHost)
	require.Equal(t, 90*time.Second, transport.IdleConnTimeout)
	require.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
	require.Equal(t, time.Second, transport.ExpectContinueTimeout)
}

func TestNewAppliesMiddlewareInDeclaredOrder(t *testing.T) {
	t.Parallel()

	var order []string

	client := New(Config{}, func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			order = append(order, "first")
			return next.RoundTrip(r)
		})
	}, func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			order = append(order, "second")
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       http.NoBody,
				Request:    r,
			}, nil
		})
	})

	_, err := client.Transport.RoundTrip((&http.Request{Header: make(http.Header)}).WithContext(t.Context()))
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, order)
}
