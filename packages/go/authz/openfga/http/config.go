package http

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout   = 5 * time.Second
	defaultMaxRetry  = 3
	defaultRetryWait = 100 * time.Millisecond
)

type AuthMethod string

const (
	AuthMethodNone              AuthMethod = "none"
	AuthMethodAPIToken          AuthMethod = "api_token"
	AuthMethodClientCredentials AuthMethod = "client_credentials"
)

type Config struct {
	APIURL               string
	StoreID              string
	AuthorizationModelID string
	Credentials          CredentialsConfig
	Timeout              time.Duration
	Retry                RetryConfig
	DefaultHeaders       map[string]string
}

type CredentialsConfig struct {
	Method       AuthMethod
	APIToken     string
	ClientID     string
	ClientSecret string
	TokenIssuer  string
	Audience     string
	Scopes       string
}

type RetryConfig struct {
	MaxRetries int
	MinWait    time.Duration
}

func DefaultConfig() Config {
	return Config{
		Credentials: CredentialsConfig{Method: AuthMethodNone},
		Timeout:     defaultTimeout,
		Retry: RetryConfig{
			MaxRetries: defaultMaxRetry,
			MinWait:    defaultRetryWait,
		},
	}
}

func (c Config) Validate() error {
	apiURL := strings.TrimSpace(c.APIURL)
	if apiURL == "" {
		return errors.New("openfga api url is required")
	}
	parsedURL, err := url.Parse(apiURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return errors.New("openfga api url must be an absolute http or https url")
	}
	switch {
	case strings.TrimSpace(c.StoreID) == "":
		return errors.New("openfga store id is required")
	case strings.TrimSpace(c.AuthorizationModelID) == "":
		return errors.New("openfga authorization model id is required")
	case c.Timeout <= 0:
		return errors.New("openfga timeout must be positive")
	}
	if err := c.Credentials.Validate(); err != nil {
		return err
	}
	return c.Retry.Validate()
}

func (c CredentialsConfig) Validate() error {
	method := c.Method
	if method == "" {
		method = AuthMethodNone
	}
	switch method {
	case AuthMethodNone:
		return nil
	case AuthMethodAPIToken:
		if strings.TrimSpace(c.APIToken) == "" {
			return errors.New("openfga api token is required for api token authentication")
		}
		return nil
	case AuthMethodClientCredentials:
		switch {
		case strings.TrimSpace(c.ClientID) == "":
			return errors.New("openfga client id is required for client credentials authentication")
		case strings.TrimSpace(c.ClientSecret) == "":
			return errors.New("openfga client secret is required for client credentials authentication")
		case strings.TrimSpace(c.TokenIssuer) == "":
			return errors.New("openfga token issuer is required for client credentials authentication")
		default:
			return nil
		}
	default:
		return errors.New("openfga authentication method is unsupported")
	}
}

func (c RetryConfig) Validate() error {
	switch {
	case c.MaxRetries < 0 || c.MaxRetries > 15:
		return errors.New("openfga max retries must be between 0 and 15")
	case c.MinWait < time.Millisecond:
		return errors.New("openfga retry min wait must be at least one millisecond")
	default:
		return nil
	}
}
