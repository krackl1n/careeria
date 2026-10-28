package middleware

import "net/http"

func UserAgent(value string) func(http.RoundTripper) http.RoundTripper {
	if value == "" {
		panic("http client user agent middleware: empty user agent")
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("User-Agent") != "" {
				return next.RoundTrip(r)
			}

			clone := cloneRequest(r)
			clone.Header.Set("User-Agent", value)

			return next.RoundTrip(clone)
		})
	}
}
