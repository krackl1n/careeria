package config

import "strings"

var initialisms = []string{
	"HTTPS",
	"GRPC",
	"HTTP",
	"HSTS",
	"JSON",
	"OTLP",
	"SASL",
	"SSL",
	"SMTP",
	"SQL",
	"SSH",
	"MTLS",
	"TLS",
	"TTL",
	"UUID",
	"GUID",
	"URL",
	"URI",
	"JWT",
	"JWKS",
	"JWK",
	"PEM",
	"DSN",
	"DNS",
	"TCP",
	"UDP",
	"RPC",
	"OIDC",
	"OAuth",
	"API",
	"CPU",
	"RAM",
	"XML",
	"HTML",
	"CIDR",
	"CORS",
	"IPv6",
	"IPv4",
	"IP",
	"ID",
	"CA",
	"DB",
	"IO",
}

func snakeCase(value string) string {
	var words []string

	for len(value) > 0 {
		if initialism := findInitialism(value); initialism != "" {
			words = appendWord(words, strings.ToLower(initialism))
			value = value[len(initialism):]
			continue
		}

		end := 1

		for end < len(value) && !isUpper(value[end]) {
			end++
		}

		words = appendWord(words, strings.ToLower(value[:end]))
		value = value[end:]
	}

	return strings.Join(words, "_")
}

func findInitialism(value string) string {
	for _, initialism := range initialisms {
		if strings.HasPrefix(value, initialism) {
			return initialism
		}
	}

	return ""
}

func appendWord(words []string, word string) []string {
	if word == "" {
		return words
	}

	if isDigits(word) && len(words) > 0 {
		words[len(words)-1] += word
		return words
	}

	return append(words, word)
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}

	return true
}

func isUpper(value byte) bool {
	return value >= 'A' && value <= 'Z'
}
