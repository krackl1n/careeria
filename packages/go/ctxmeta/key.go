package ctxmeta

type key uint8

const (
	requestIDKey key = iota
	correlationIDKey
	messageIDKey
	causationIDKey
	principalKey
	tenantIDKey
)
