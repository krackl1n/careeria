package grpcintegration

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcclient "github.com/krackl1n/careeria/grpc/client"
	grpcclientinterceptor "github.com/krackl1n/careeria/grpc/client/interceptor"
)

func TestNewDialerRequiresMetrics(t *testing.T) {
	dialer, err := NewDialer(nil)

	require.Nil(t, dialer)
	require.EqualError(t, err, "integration gRPC client metrics are required")
}

func TestDialValidatesPeerService(t *testing.T) {
	dialer := newTestDialer(t)

	connection, err := dialer.Dial(grpcclient.Config{}, " ")

	require.Nil(t, connection)
	require.EqualError(t, err, "integration gRPC peer service is required")
}

func TestDialAppliesGatewayAuthenticationPolicy(t *testing.T) {
	dialer := newTestDialer(t)
	connection, err := dialer.Dial(grpcclient.Config{
		Target:   "127.0.0.1:1",
		Insecure: true,
	}, "identity")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })

	err = connection.Invoke(
		context.Background(),
		"/careeria.identity.v1.OAuthService/GetUserInfo",
		nil,
		nil,
	)

	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func newTestDialer(t *testing.T) *Dialer {
	t.Helper()

	metrics, err := grpcclientinterceptor.NewMetrics(
		grpcclientinterceptor.MetricsConfig{},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	dialer, err := NewDialer(metrics)
	require.NoError(t, err)
	return dialer
}
