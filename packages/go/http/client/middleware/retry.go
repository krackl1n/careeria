package middleware

import (
	"context"
	"net/http"
)

type RetryFunc func(
	ctx context.Context,
	req *http.Request,
	attempt func(context.Context, *http.Request) (*http.Response, error),
) (*http.Response, error)

func Retry(retry RetryFunc) func(http.RoundTripper) http.RoundTripper {
	if retry == nil {
		panic("http client retry middleware: nil retry function")
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return retry(
				r.Context(),
				r,
				func(ctx context.Context, req *http.Request) (*http.Response, error) {
					if req == nil {
						req = r
					}

					if req.Context() != ctx {
						req = req.Clone(ctx)
					}

					return next.RoundTrip(req)
				},
			)
		})
	}
}
