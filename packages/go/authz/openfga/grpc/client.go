package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/krackl1n/careeria/authz"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

const tenantContext = "tenant_id"

var (
	ErrNilConnection   = errors.New("openfga: grpc connection is nil")
	ErrUnavailable     = errors.New("openfga: grpc client is unavailable")
	ErrInvalidResponse = errors.New("openfga: invalid grpc response")
)

type Client struct {
	storeID              string
	authorizationModelID string
	timeout              time.Duration
	openfga              openfgav1.OpenFGAServiceClient
}

var _ authz.Authorizer = (*Client)(nil)

func New(config Config, connection googlegrpc.ClientConnInterface) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate OpenFGA gRPC config: %w", err)
	}
	if connection == nil {
		return nil, ErrNilConnection
	}
	return &Client{
		storeID:              strings.TrimSpace(config.StoreID),
		authorizationModelID: strings.TrimSpace(config.AuthorizationModelID),
		timeout:              config.Timeout,
		openfga:              openfgav1.NewOpenFGAServiceClient(connection),
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.openfga == nil {
		return ErrUnavailable
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if _, err := c.openfga.GetStore(callCtx, &openfgav1.GetStoreRequest{StoreId: c.storeID}); err != nil {
		return fmt.Errorf("ping OpenFGA over gRPC: %w", err)
	}
	return nil
}

func (c *Client) Check(ctx context.Context, access authz.Access) (authz.Decision, error) {
	if c == nil || c.openfga == nil {
		return authz.Decision{}, ErrUnavailable
	}
	if err := access.Validate(); err != nil {
		return authz.Decision{}, fmt.Errorf("validate access: %w", err)
	}

	conditionContext, err := structpb.NewStruct(withTenant(access.Tenant.ID, access.ConditionContext))
	if err != nil {
		return authz.Decision{}, fmt.Errorf("encode OpenFGA condition context: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.openfga.Check(callCtx, &openfgav1.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		TupleKey: &openfgav1.CheckRequestTupleKey{
			User:     authz.User(access.Actor.Type, access.Actor.ID).String(),
			Relation: access.Permission,
			Object:   access.Resource.String(),
		},
		Context:     conditionContext,
		Consistency: consistency(access.Consistency),
	})
	if err != nil {
		return authz.Decision{}, fmt.Errorf("check OpenFGA relation over gRPC: %w", err)
	}
	if response == nil {
		return authz.Decision{}, ErrInvalidResponse
	}
	return authz.Decision{Allowed: response.GetAllowed()}, nil
}

func (c *Client) Require(ctx context.Context, access authz.Access) error {
	decision, err := c.Check(ctx, access)
	if err != nil {
		return err
	}
	if decision.Allowed {
		return nil
	}
	return &authz.DeniedError{Access: access}
}

func withTenant(tenantID string, values map[string]any) map[string]any {
	result := make(map[string]any, len(values)+1)
	for key, value := range values {
		result[key] = value
	}
	result[tenantContext] = tenantID
	return result
}

func consistency(value authz.Consistency) openfgav1.ConsistencyPreference {
	switch value {
	case authz.ConsistencyMinimizeLatency:
		return openfgav1.ConsistencyPreference_MINIMIZE_LATENCY
	case authz.ConsistencyHigherConsistency:
		return openfgav1.ConsistencyPreference_HIGHER_CONSISTENCY
	default:
		return openfgav1.ConsistencyPreference_UNSPECIFIED
	}
}
