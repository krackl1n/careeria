package oauthclient

import (
	"fmt"
	"net/url"
	"strings"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

func parseExactURI(value, field string) (string, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", fmt.Errorf("%s must be a non-empty URI without whitespace", field)
	}

	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Scheme == "" {
		return "", fmt.Errorf("%s must be an absolute URI", field)
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("%s must not contain a fragment", field)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%s must not contain user information", field)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "https":
		if parsed.Host == "" {
			return "", fmt.Errorf("%s must contain a host", field)
		}
	case "http":
		if parsed.Host == "" || !isLoopbackHost(parsed.Hostname()) {
			return "", fmt.Errorf("%s may use http only for a loopback host", field)
		}
	default:
		// RFC 8252 allows private-use URI schemes for native applications. Requiring
		// a reverse-domain style scheme excludes browser-executable schemes such as
		// javascript: and data: while retaining com.example.app:/oauth/callback.
		if !strings.Contains(scheme, ".") {
			return "", fmt.Errorf("%s has an unsupported URI scheme", field)
		}
	}

	return value, nil
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
