package runtime

import "context"

type Component interface {
	Name() string
	Start() error
	Shutdown(context.Context) error
}

type Result struct {
	Name string
	Err  error
}
