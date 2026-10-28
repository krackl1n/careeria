package apperr

import "fmt"

type Type uint8

const (
	TypeUnknown Type = iota
	TypeInvalidArgument
	TypeUnauthorized
	TypeForbidden
	TypeNotFound
	TypeConflict
	TypeUnavailable
	TypeInternal
)

func (t Type) String() string {
	switch t {
	case TypeInvalidArgument:
		return "invalid_argument"
	case TypeUnauthorized:
		return "unauthorized"
	case TypeForbidden:
		return "forbidden"
	case TypeNotFound:
		return "not_found"
	case TypeConflict:
		return "conflict"
	case TypeUnavailable:
		return "unavailable"
	case TypeInternal:
		return "internal"
	default:
		return "unknown"
	}
}

func (t Type) Valid() bool {
	switch t {
	case TypeInvalidArgument,
		TypeUnauthorized,
		TypeForbidden,
		TypeNotFound,
		TypeConflict,
		TypeUnavailable,
		TypeInternal:
		return true
	default:
		return false
	}
}

func (t Type) GoString() string {
	return fmt.Sprintf("apperr.Type(%q)", t.String())
}
