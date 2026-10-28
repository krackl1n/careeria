package grpcintegration

import (
	"errors"
	"fmt"
	"strings"

	grpcclient "github.com/krackl1n/careeria/grpc/client"
	grpcclientinterceptor "github.com/krackl1n/careeria/grpc/client/interceptor"
	"github.com/krackl1n/careeria/services/api-gateway/internal/authn"
	"google.golang.org/grpc"
)

type Dialer struct {
	metrics *grpcclientinterceptor.Metrics
}

func NewDialer(metrics *grpcclientinterceptor.Metrics) (*Dialer, error) {
	if metrics == nil {
		return nil, errors.New("integration gRPC client metrics are required")
	}

	return &Dialer{metrics: metrics}, nil
}

func (d *Dialer) Dial(config grpcclient.Config, peerService string) (*grpc.ClientConn, error) {
	if strings.TrimSpace(peerService) == "" {
		return nil, errors.New("integration gRPC peer service is required")
	}

	connection, err := grpcclient.New(
		config,
		grpc.WithChainUnaryInterceptor(
			grpcclientinterceptor.UnaryRequestID(),
			d.metrics.UnaryClientInterceptor(peerService),
			authn.UnaryPolicy(),
			grpcclientinterceptor.UnaryValidation(),
		),
		grpc.WithChainStreamInterceptor(
			grpcclientinterceptor.StreamRequestID(),
			d.metrics.StreamClientInterceptor(peerService),
			authn.StreamPolicy(),
			grpcclientinterceptor.StreamValidation(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s integration: %w", peerService, err)
	}

	return connection, nil
}
