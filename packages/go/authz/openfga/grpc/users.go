package grpc

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/authz"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func (c *Client) ListUsers(ctx context.Context, request authz.ListUsersRequest) ([]authz.Subject, error) {
	if c == nil || c.openfga == nil {
		return nil, ErrUnavailable
	}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validate list users: %w", err)
	}
	conditionContext, err := structpb.NewStruct(withTenant(request.Tenant.ID, request.ConditionContext))
	if err != nil {
		return nil, fmt.Errorf("encode OpenFGA condition context: %w", err)
	}
	contextualTuples := make([]*openfgav1.TupleKey, 0, len(request.ContextualTuples))
	for _, tuple := range request.ContextualTuples {
		contextualTuples = append(contextualTuples, &openfgav1.TupleKey{User: tuple.Subject.String(), Relation: tuple.Relation, Object: tuple.Object.String()})
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.openfga.ListUsers(callCtx, &openfgav1.ListUsersRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		Object:               &openfgav1.Object{Type: request.Object.Type, Id: request.Object.ID},
		Relation:             request.Relation,
		UserFilters:          []*openfgav1.UserTypeFilter{{Type: request.SubjectType}},
		ContextualTuples:     contextualTuples,
		Context:              conditionContext,
		Consistency:          consistency(request.Consistency),
	})
	if err != nil {
		return nil, fmt.Errorf("list OpenFGA users over gRPC: %w", err)
	}
	if response == nil {
		return nil, ErrInvalidResponse
	}
	users := make([]authz.Subject, 0, len(response.GetUsers()))
	for index, value := range response.GetUsers() {
		object := value.GetObject()
		if object == nil || object.GetType() != request.SubjectType {
			return nil, fmt.Errorf("parse OpenFGA user %d: unexpected user type", index)
		}
		subject := authz.User(object.GetType(), object.GetId())
		if err := subject.Validate(); err != nil {
			return nil, fmt.Errorf("parse OpenFGA user %d: %w", index, err)
		}
		users = append(users, subject)
	}
	return users, nil
}
