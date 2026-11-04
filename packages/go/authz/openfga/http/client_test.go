package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krackl1n/careeria/authz"
	"github.com/stretchr/testify/require"
)

func TestRequireTenantPermission(t *testing.T) {
	t.Parallel()

	allowed := true
	server := httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		require.Equal(t, "/stores/01FQH7V8BEG3GPQW93KTRFR8JB/check", request.URL.Path)
		require.Equal(t, "Bearer token", request.Header.Get("Authorization"))
		var body struct {
			TupleKey struct {
				User     string `json:"user"`
				Relation string `json:"relation"`
				Object   string `json:"object"`
			} `json:"tuple_key"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "user:42", body.TupleKey.User)
		require.Equal(t, "can_create_meeting", body.TupleKey.Relation)
		require.Equal(t, "tenant:7", body.TupleKey.Object)
		writer.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(writer).Encode(map[string]any{"allowed": allowed}))
	}))
	defer server.Close()

	config := validConfig(server.URL)
	config.Credentials = CredentialsConfig{Method: AuthMethodAPIToken, APIToken: "token"}
	client, err := NewWithClient(config, server.Client())
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

	server := httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		require.Equal(t, "/stores/01FQH7V8BEG3GPQW93KTRFR8JB/check", request.URL.Path)
		var body struct {
			TupleKey struct {
				User     string `json:"user"`
				Relation string `json:"relation"`
				Object   string `json:"object"`
			} `json:"tuple_key"`
			Context map[string]any `json:"context"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "user:42", body.TupleKey.User)
		require.Equal(t, "can_update", body.TupleKey.Relation)
		require.Equal(t, "meeting:10", body.TupleKey.Object)
		require.Equal(t, "7", body.Context[tenantContext])
		writer.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(writer).Encode(map[string]any{"allowed": true}))
	}))
	defer server.Close()

	client, err := NewWithClient(validConfig(server.URL), server.Client())
	require.NoError(t, err)
	decision, err := client.Check(t.Context(), authz.Access{
		Actor:      authz.Actor{Type: "user", ID: "42"},
		Tenant:     authz.Tenant{ID: "7"},
		Permission: "can_update",
		Resource:   authz.Resource("meeting", "10"),
	})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
}

func TestRelationshipsAndPing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/stores/01FQH7V8BEG3GPQW93KTRFR8JB":
			_, _ = writer.Write([]byte(`{"id":"01FQH7V8BEG3GPQW93KTRFR8JB","name":"careeria"}`))
		case "/stores/01FQH7V8BEG3GPQW93KTRFR8JB/write":
			_, _ = writer.Write([]byte(`{}`))
		case "/stores/01FQH7V8BEG3GPQW93KTRFR8JB/list-objects":
			_, _ = writer.Write([]byte(`{"objects":["meeting:10"]}`))
		default:
			nethttp.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewWithClient(validConfig(server.URL), server.Client())
	require.NoError(t, err)
	require.NoError(t, client.Ping(t.Context()))
	require.NoError(t, client.Write(t.Context(), authz.WriteRequest{Writes: []authz.Tuple{{
		Subject: authz.User("user", "42"), Relation: "owner", Object: authz.Resource("meeting", "10"),
	}}}))
	objects, err := client.ListObjects(t.Context(), authz.ListObjectsRequest{
		Subject: authz.User("user", "42"), Tenant: authz.Tenant{ID: "7"},
		Relation: "can_view", ObjectType: "meeting",
	})
	require.NoError(t, err)
	require.Equal(t, []authz.Object{authz.Resource("meeting", "10")}, objects)
}

func TestClientAndConfigValidation(t *testing.T) {
	t.Parallel()

	var client *Client
	_, err := client.Check(context.Background(), authz.Access{})
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = NewWithClient(validConfig("http://openfga:8080"), nil)
	require.True(t, errors.Is(err, ErrNilClient))

	config := DefaultConfig()
	require.Equal(t, CredentialsConfig{Method: AuthMethodNone}, config.Credentials)
	require.Equal(t, 5*time.Second, config.Timeout)
	config.APIURL = "openfga:8080"
	config.StoreID = "01FQH7V8BEG3GPQW93KTRFR8JB"
	config.AuthorizationModelID = "01HVMMBCMGZNT3SED4Z17ECXCA"
	require.EqualError(t, config.Validate(), "openfga api url must be an absolute http or https url")
}

func validConfig(apiURL string) Config {
	config := DefaultConfig()
	config.APIURL = apiURL
	config.StoreID = "01FQH7V8BEG3GPQW93KTRFR8JB"
	config.AuthorizationModelID = "01HVMMBCMGZNT3SED4Z17ECXCA"
	return config
}
