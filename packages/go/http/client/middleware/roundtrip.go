package middleware

import "net/http"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func cloneRequest(r *http.Request) *http.Request {
	clone := r.Clone(r.Context())

	if r.Header != nil {
		clone.Header = r.Header.Clone()
	}

	return clone
}
