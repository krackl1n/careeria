package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krackl1n/careeria/authz"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/stretchr/testify/require"
	googlegrpc "google.golang.org/grpc"
)

type connectionStub struct {
	invoke func(string, any, any) error
}

func (s *connectionStub) Invoke(
	_ context.Context,
	method string,
	request any,
	response any,
	_ ...googlegrpc.CallOption,
) error {
	return s.invoke(method, request, response)
}

func (s *connectionStub) NewStream(
	context.Context,
	*googlegrpc.StreamDesc,
	string,
	...googlegrpc.CallOption,
) (googlegrpc.ClientStream, error) {
	return nil, errors.New("streaming is not expected")
}

func TestRequireTenantPermission(t *testing.T) {
	t.Parallel()

	allowed := true
	client, err := New(validConfig(), &connectionStub{invoke: func(method string, input, output any) error {
		require.Equal(t, openfgav1.OpenFGAService_Check_FullMethodName, method)
		request := input.(*openfgav1.CheckRequest)
		require.Equal(t, "store-id", request.GetStoreId())
		require.Equal(t, "model-id", request.GetAuthorizationModelId())
		require.Equal(t, "user:42", request.GetTupleKey().GetUser())
		require.Equal(t, "can_create_meeting", request.GetTupleKey().GetRelation())
		require.Equal(t, "tenant:7", request.GetTupleKey().GetObject())
		require.Equal(t, "7", request.GetContext().AsMap()[tenantContext])
		output.(*openfgav1.CheckResponse).Allowed = allowed
		return nil
	}})
	require.NoError(t, err)

	access := authz.Access{
		Actor:      authz.Actor{Type: "user", ID: "42"},
		Tenant:     authz.Tenant{ID: "7"},
		Permission: "can_create_meeting",
		Resource:   authz.TenantObject("7"),
	}
	require.NoError(t, client.Require(t.Context(), access))
	allowed = false
	require.ErrorIs(t, client.Require(t.Context(), access), authz.ErrDenied)
}

func TestCheckResourcePermission(t *testing.T) {
	t.Parallel()

	client, err := New(validConfig(), &connectionStub{invoke: func(method string, input, output any) error {
		require.Equal(t, openfgav1.OpenFGAService_Check_FullMethodName, method)
		request := input.(*openfgav1.CheckRequest)
		require.Equal(t, openfgav1.ConsistencyPreference_HIGHER_CONSISTENCY, request.GetConsistency())
		require.Equal(t, "user:42", request.GetTupleKey().GetUser())
		require.Equal(t, "can_update", request.GetTupleKey().GetRelation())
		require.Equal(t, "meeting:10", request.GetTupleKey().GetObject())
		require.Equal(t, "7", request.GetContext().AsMap()[tenantContext])
		output.(*openfgav1.CheckResponse).Allowed = true
		return nil
	}})
	require.NoError(t, err)

	decision, err := client.Check(t.Context(), authz.Access{
		Actor:       authz.Actor{Type: "user", ID: "42"},
		Tenant:      authz.Tenant{ID: "7"},
		Permission:  "can_update",
		Resource:    authz.Resource("meeting", "10"),
		Consistency: authz.ConsistencyHigherConsistency,
	})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
}

func TestNewRequiresConnection(t *testing.T) {
	t.Parallel()

	_, err := New(validConfig(), nil)
	require.ErrorIs(t, err, ErrNilConnection)
}

func TestListObjects(t *testing.T) {
	t.Parallel()

	client, err := New(validConfig(), &connectionStub{invoke: func(method string, input, output any) error {
		require.Equal(t, openfgav1.OpenFGAService_ListObjects_FullMethodName, method)
		request := input.(*openfgav1.ListObjectsRequest)
		require.Equal(t, "user:42", request.GetUser())
		require.Equal(t, "meeting_view", request.GetRelation())
		require.Equal(t, "meeting", request.GetType())
		require.Equal(t, "7", request.GetContext().AsMap()[tenantContext])
		output.(*openfgav1.ListObjectsResponse).Objects = []string{"meeting:10", "meeting:11"}
		return nil
	}})
	require.NoError(t, err)

	objects, err := client.ListObjects(t.Context(), authz.ListObjectsRequest{
		Subject:    authz.User("user", "42"),
		Tenant:     authz.Tenant{ID: "7"},
		Relation:   "meeting_view",
		ObjectType: "meeting",
	})
	require.NoError(t, err)
	require.Equal(t, []authz.Object{authz.Resource("meeting", "10"), authz.Resource("meeting", "11")}, objects)
}

func TestListUsers(t *testing.T) {
	t.Parallel()

	client, err := New(validConfig(), &connectionStub{invoke: func(method string, input, output any) error {
		require.Equal(t, openfgav1.OpenFGAService_ListUsers_FullMethodName, method)
		request := input.(*openfgav1.ListUsersRequest)
		require.Equal(t, "document", request.GetObject().GetType())
		require.Equal(t, "10", request.GetObject().GetId())
		require.Equal(t, "document_approve", request.GetRelation())
		require.Equal(t, "user", request.GetUserFilters()[0].GetType())
		require.Equal(t, "7", request.GetContext().AsMap()[tenantContext])
		output.(*openfgav1.ListUsersResponse).Users = []*openfgav1.User{
			{User: &openfgav1.User_Object{Object: &openfgav1.Object{Type: "user", Id: "42"}}},
			{User: &openfgav1.User_Object{Object: &openfgav1.Object{Type: "user", Id: "43"}}},
		}
		return nil
	}})
	require.NoError(t, err)

	users, err := client.ListUsers(t.Context(), authz.ListUsersRequest{
		Object: authz.Resource("document", "10"), Tenant: authz.Tenant{ID: "7"}, Relation: "document_approve", SubjectType: "user",
	})
	require.NoError(t, err)
	require.Equal(t, []authz.Subject{authz.User("user", "42"), authz.User("user", "43")}, users)
}

func validConfig() Config {
	return Config{StoreID: "store-id", AuthorizationModelID: "model-id", Timeout: time.Second}
}
