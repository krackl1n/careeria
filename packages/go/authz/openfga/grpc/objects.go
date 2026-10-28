package grpc

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/authz"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func (c *Client) ListObjects(ctx context.Context, request authz.ListObjectsRequest) ([]authz.Object, error) {
	if c == nil || c.openfga == nil {
		return nil, ErrUnavailable
	}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validate list objects: %w", err)
	}

	conditionContext, err := structpb.NewStruct(withTenant(request.Tenant.ID, request.ConditionContext))
	if err != nil {
		return nil, fmt.Errorf("encode OpenFGA condition context: %w", err)
	}

	contextualTuples := make([]*openfgav1.TupleKey, 0, len(request.ContextualTuples))
	for index, tuple := range request.ContextualTuples {
		value := &openfgav1.TupleKey{
			User:     tuple.Subject.String(),
			Relation: tuple.Relation,
			Object:   tuple.Object.String(),
		}
		if tuple.Condition != nil {
			condition, err := structpb.NewStruct(tuple.Condition.Context)
			if err != nil {
				return nil, fmt.Errorf("encode OpenFGA contextual tuple %d condition: %w", index, err)
			}
			value.Condition = &openfgav1.RelationshipCondition{
				Name:    tuple.Condition.Name,
				Context: condition,
			}
		}
		contextualTuples = append(contextualTuples, value)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.openfga.ListObjects(callCtx, &openfgav1.ListObjectsRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		Type:                 request.ObjectType,
		Relation:             request.Relation,
		User:                 request.Subject.String(),
		Context:              conditionContext,
		ContextualTuples:     &openfgav1.ContextualTupleKeys{TupleKeys: contextualTuples},
		Consistency:          consistency(request.Consistency),
	})
	if err != nil {
		return nil, fmt.Errorf("list OpenFGA objects over gRPC: %w", err)
	}
	if response == nil {
		return nil, ErrInvalidResponse
	}

	objects := make([]authz.Object, 0, len(response.GetObjects()))
	for index, value := range response.GetObjects() {
		object, err := authz.ParseObject(value)
		if err != nil {
			return nil, fmt.Errorf("parse OpenFGA object %d: %w", index, err)
		}
		if object.Type != request.ObjectType {
			return nil, fmt.Errorf("parse OpenFGA object %d: unexpected type %q", index, object.Type)
		}
		objects = append(objects, object)
	}
	return objects, nil
}
