package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const ClientIPHeader = "X-Client-IP"

type RealIPConfig struct {
	TrustedProxies []netip.Prefix
}

func RealIP(config RealIPConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("real ip middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del(ClientIPHeader)

			clientIP := resolveClientIP(r, config.TrustedProxies)
			if clientIP != "" {
				r.Header.Set(ClientIPHeader, clientIP)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ClientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if clientIP := r.Header.Get(ClientIPHeader); clientIP != "" {
		return clientIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}

func resolveClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	remoteIP, ok := parseRemoteIP(r.RemoteAddr)
	if !ok {
		return ""
	}

	if !isTrustedProxy(remoteIP, trustedProxies) {
		return remoteIP.String()
	}

	chain := parseForwardedFor(r.Header.Get("X-Forwarded-For"))
	if len(chain) == 0 {
		return remoteIP.String()
	}

	for i := len(chain) - 1; i >= 0; i-- {
		if !isTrustedProxy(chain[i], trustedProxies) {
			return chain[i].String()
		}
	}

	return chain[0].String()
}

func parseRemoteIP(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}

	return ip, true
}

func parseForwardedFor(value string) []netip.Addr {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]netip.Addr, 0, len(parts))

	for _, part := range parts {
		ip, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil {
			continue
		}

		result = append(result, ip)
	}

	return result
}

func isTrustedProxy(ip netip.Addr, trustedProxies []netip.Prefix) bool {
	for _, prefix := range trustedProxies {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}
