package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/krackl1n/careeria/authz"
	sdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
)

const (
	userAgent     = "careeria-authz/1.0"
	tenantContext = "tenant_id"
)

var (
	ErrNilClient       = errors.New("openfga: http client is nil")
	ErrUnavailable     = errors.New("openfga: http client is unavailable")
	ErrInvalidResponse = errors.New("openfga: invalid http response")
)

type Client struct {
	openfga *fgaclient.OpenFgaClient
}

var (
	_ authz.Authorizer        = (*Client)(nil)
	_ authz.RelationshipStore = (*Client)(nil)
)

func New(config Config) (*Client, error) {
	return NewWithClient(config, &nethttp.Client{Timeout: config.Timeout})
}

func NewWithClient(config Config, httpClient *nethttp.Client) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate OpenFGA HTTP config: %w", err)
	}
	if httpClient == nil {
		return nil, ErrNilClient
	}

	baseClient := *httpClient
	if baseClient.Timeout == 0 {
		baseClient.Timeout = config.Timeout
	}
	client, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:               strings.TrimRight(strings.TrimSpace(config.APIURL), "/"),
		StoreId:              strings.TrimSpace(config.StoreID),
		AuthorizationModelId: strings.TrimSpace(config.AuthorizationModelID),
		Credentials:          credentialsFromConfig(config.Credentials),
		DefaultHeaders:       cloneHeaders(config.DefaultHeaders),
		UserAgent:            userAgent,
		HTTPClient:           &baseClient,
		RetryParams: &sdk.RetryParams{
			MaxRetry:    config.Retry.MaxRetries,
			MinWaitInMs: int(config.Retry.MinWait.Milliseconds()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create OpenFGA HTTP client: %w", err)
	}
	return &Client{openfga: client}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.openfga == nil {
		return ErrUnavailable
	}
	if _, err := c.openfga.GetStore(ctx).Execute(); err != nil {
		return fmt.Errorf("ping OpenFGA over HTTP: %w", err)
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

	conditionContext := withTenant(access.Tenant.ID, access.ConditionContext)
	response, err := c.openfga.Check(ctx).
		Body(fgaclient.ClientCheckRequest{
			User:     authz.User(access.Actor.Type, access.Actor.ID).String(),
			Relation: access.Permission,
			Object:   access.Resource.String(),
			Context:  &conditionContext,
		}).
		Options(fgaclient.ClientCheckOptions{Consistency: consistency(access.Consistency)}).
		Execute()
	if err != nil {
		return authz.Decision{}, fmt.Errorf("check OpenFGA relation over HTTP: %w", err)
	}
	if response == nil || response.Allowed == nil {
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

func consistency(value authz.Consistency) *sdk.ConsistencyPreference {
	var result sdk.ConsistencyPreference
	switch value {
	case authz.ConsistencyMinimizeLatency:
		result = sdk.CONSISTENCYPREFERENCE_MINIMIZE_LATENCY
	case authz.ConsistencyHigherConsistency:
		result = sdk.CONSISTENCYPREFERENCE_HIGHER_CONSISTENCY
	default:
		return nil
	}
	return &result
}

func credentialsFromConfig(config CredentialsConfig) *credentials.Credentials {
	method := config.Method
	if method == "" {
		method = AuthMethodNone
	}
	return &credentials.Credentials{
		Method: credentials.CredentialsMethod(method),
		Config: &credentials.Config{
			ApiToken:                        config.APIToken,
			ClientCredentialsClientId:       config.ClientID,
			ClientCredentialsClientSecret:   config.ClientSecret,
			ClientCredentialsApiTokenIssuer: config.TokenIssuer,
			ClientCredentialsApiAudience:    config.Audience,
			ClientCredentialsScopes:         config.Scopes,
		},
	}
}

func cloneHeaders(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
