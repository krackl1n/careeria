package runtime

import "errors"

var (
	ErrNoComponents           = errors.New("runtime: at least one component is required")
	ErrNilComponent           = errors.New("runtime: component is required")
	ErrEmptyComponentName     = errors.New("runtime: component name is required")
	ErrDuplicateComponentName = errors.New("runtime: component name must be unique")
	ErrNilContext             = errors.New("runtime: context is required")
	ErrNilContextRunner       = errors.New("runtime: context runner is required")
)
