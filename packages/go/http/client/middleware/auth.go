package middleware

import (
	"context"
	"net/http"
)

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

func BearerToken(provider TokenProvider) func(http.RoundTripper) http.RoundTripper {
	if provider == nil {
		panic("http client bearer token middleware: nil token provider")
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			token, err := provider.Token(r.Context())
			if err != nil {
				return nil, err
			}

			if token == "" {
				return next.RoundTrip(r)
			}

			clone := cloneRequest(r)
			clone.Header.Set("Authorization", "Bearer "+token)

			return next.RoundTrip(clone)
		})
	}
}

func APIKey(header string, provider func(context.Context) (string, error)) func(http.RoundTripper) http.RoundTripper {
	if header == "" {
		panic("http client api key middleware: empty header")
	}

	if provider == nil {
		panic("http client api key middleware: nil provider")
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			value, err := provider(r.Context())
			if err != nil {
				return nil, err
			}

			if value == "" {
				return next.RoundTrip(r)
			}

			clone := cloneRequest(r)
			clone.Header.Set(header, value)

			return next.RoundTrip(clone)
		})
	}
}
