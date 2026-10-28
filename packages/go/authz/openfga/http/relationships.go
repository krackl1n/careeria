package http

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/authz"
	sdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
)

func (c *Client) Write(ctx context.Context, request authz.WriteRequest) error {
	if c == nil || c.openfga == nil {
		return ErrUnavailable
	}
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate relationship mutation: %w", err)
	}
	body := fgaclient.ClientWriteRequest{Writes: tuples(request.Writes), Deletes: deletedTuples(request.Deletes)}
	options := fgaclient.ClientWriteOptions{Conflict: fgaclient.ClientWriteConflictOptions{
		OnDuplicateWrites: duplicateWriteBehavior(request.IgnoreDuplicateWrites),
		OnMissingDeletes:  missingDeleteBehavior(request.IgnoreMissingDeletes),
	}}
	if _, err := c.openfga.Write(ctx).Body(body).Options(options).Execute(); err != nil {
		return fmt.Errorf("write OpenFGA relationships: %w", err)
	}
	return nil
}

func (c *Client) ListObjects(ctx context.Context, request authz.ListObjectsRequest) ([]authz.Object, error) {
	if c == nil || c.openfga == nil {
		return nil, ErrUnavailable
	}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validate list objects: %w", err)
	}
	conditionContext := withTenant(request.Tenant.ID, request.ConditionContext)
	response, err := c.openfga.ListObjects(ctx).
		Body(fgaclient.ClientListObjectsRequest{
			User:             request.Subject.String(),
			Relation:         request.Relation,
			Type:             request.ObjectType,
			Context:          &conditionContext,
			ContextualTuples: tuples(request.ContextualTuples),
		}).
		Options(fgaclient.ClientListObjectsOptions{Consistency: consistency(request.Consistency)}).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("list OpenFGA objects: %w", err)
	}
	if response == nil {
		return nil, ErrInvalidResponse
	}

	objects := make([]authz.Object, 0, len(response.Objects))
	for index, value := range response.Objects {
		object, err := authz.ParseObject(value)
		if err != nil {
			return nil, fmt.Errorf("parse OpenFGA object %d: %w", index, err)
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func tuples(values []authz.Tuple) []fgaclient.ClientTupleKey {
	if len(values) == 0 {
		return nil
	}
	result := make([]fgaclient.ClientTupleKey, 0, len(values))
	for _, value := range values {
		tuple := fgaclient.ClientTupleKey{
			User:     value.Subject.String(),
			Relation: value.Relation,
			Object:   value.Object.String(),
		}
		if value.Condition != nil {
			tuple.Condition = &sdk.RelationshipCondition{Name: value.Condition.Name, Context: &value.Condition.Context}
		}
		result = append(result, tuple)
	}
	return result
}

func deletedTuples(values []authz.Tuple) []fgaclient.ClientTupleKeyWithoutCondition {
	if len(values) == 0 {
		return nil
	}
	result := make([]fgaclient.ClientTupleKeyWithoutCondition, 0, len(values))
	for _, value := range values {
		result = append(result, fgaclient.ClientTupleKeyWithoutCondition{
			User: value.Subject.String(), Relation: value.Relation, Object: value.Object.String(),
		})
	}
	return result
}

func duplicateWriteBehavior(ignore bool) fgaclient.ClientWriteRequestOnDuplicateWrites {
	if ignore {
		return fgaclient.CLIENT_WRITE_REQUEST_ON_DUPLICATE_WRITES_IGNORE
	}
	return fgaclient.CLIENT_WRITE_REQUEST_ON_DUPLICATE_WRITES_ERROR
}

func missingDeleteBehavior(ignore bool) fgaclient.ClientWriteRequestOnMissingDeletes {
	if ignore {
		return fgaclient.CLIENT_WRITE_REQUEST_ON_MISSING_DELETES_IGNORE
	}
	return fgaclient.CLIENT_WRITE_REQUEST_ON_MISSING_DELETES_ERROR
}
