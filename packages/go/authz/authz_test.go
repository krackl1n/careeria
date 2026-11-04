package authz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessValidate(t *testing.T) {
	t.Parallel()

	access := Access{
		Actor:      Actor{Type: "user", ID: "42"},
		Tenant:     Tenant{ID: "7"},
		Permission: "can_update",
		Resource:   Resource("meeting", "10"),
	}
	require.NoError(t, access.Validate())

	access.Tenant.ID = ""
	require.Error(t, access.Validate())
}

func TestDeniedError(t *testing.T) {
	t.Parallel()

	err := &DeniedError{Access: Access{
		Actor:      Actor{Type: "user", ID: "42"},
		Tenant:     Tenant{ID: "7"},
		Permission: "can_update",
		Resource:   Resource("meeting", "10"),
	}}
	require.True(t, errors.Is(err, ErrDenied))
	require.Contains(t, err.Error(), "tenant:7")
}

func TestReferences(t *testing.T) {
	t.Parallel()

	require.Equal(t, "user:42", User("user", "42").String())
	require.Equal(t, "meeting_role:10/7#assignee", Userset("meeting_role", "10/7", "assignee").String())
	require.Equal(t, "meeting:10", Resource("meeting", "10").String())
}

func TestParseObject(t *testing.T) {
	t.Parallel()

	object, err := ParseObject("document:tenant:document-id")
	require.NoError(t, err)
	require.Equal(t, Object{Type: "document", ID: "tenant:document-id"}, object)

	_, err = ParseObject("document")
	require.EqualError(t, err, "authz object must have type:id format")
}

func TestTupleValidate(t *testing.T) {
	t.Parallel()

	valid := Tuple{
		Subject:  Userset("meeting_role", "10/7", "assignee"),
		Relation: "can_update",
		Object:   Resource("meeting", "10"),
	}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name    string
		mutate  func(*Tuple)
		wantErr string
	}{
		{
			name: "subject type is invalid",
			mutate: func(tuple *Tuple) {
				tuple.Subject.Type = "meeting:role"
			},
			wantErr: "authz subject type is invalid",
		},
		{
			name: "subject id is missing",
			mutate: func(tuple *Tuple) {
				tuple.Subject.ID = " "
			},
			wantErr: "authz subject id is required",
		},
		{
			name: "relation is invalid",
			mutate: func(tuple *Tuple) {
				tuple.Relation = "can update"
			},
			wantErr: "authz relation is invalid",
		},
		{
			name: "object type is missing",
			mutate: func(tuple *Tuple) {
				tuple.Object.Type = ""
			},
			wantErr: "authz object type is required",
		},
		{
			name: "condition is invalid",
			mutate: func(tuple *Tuple) {
				tuple.Condition = &Condition{Name: " "}
			},
			wantErr: "authz condition name is invalid: authz relation is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tuple := valid
			tt.mutate(&tuple)
			require.EqualError(t, tuple.Validate(), tt.wantErr)
		})
	}
}

func TestListObjectsRequestRequiresTenant(t *testing.T) {
	t.Parallel()

	request := ListObjectsRequest{
		Subject:    User("user", "42"),
		Relation:   "can_view",
		ObjectType: "meeting",
	}
	require.EqualError(t, request.Validate(), "authz object id is required")

	request.Tenant = Tenant{ID: "7"}
	require.NoError(t, request.Validate())
}
