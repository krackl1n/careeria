package shutdown

import "context"

type Component interface {
	Name() string
	Shutdown(context.Context) error
}
