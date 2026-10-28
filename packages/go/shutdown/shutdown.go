package shutdown

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

func Run(
	parent context.Context,
	cfg Config,
	components ...Component,
) error {
	if parent == nil {
		return ErrNilContext
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := validateComponents(components); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()

	var shutdownErrors []error

	for i := len(components) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			shutdownErrors = append(
				shutdownErrors,
				fmt.Errorf("shutdown interrupted: %w", err),
			)

			break
		}

		component := components[i]

		if err := component.Shutdown(ctx); err != nil {
			shutdownErrors = append(
				shutdownErrors,
				fmt.Errorf(
					"shutdown %q: %w",
					component.Name(),
					err,
				),
			)
		}
	}

	return errors.Join(shutdownErrors...)
}

func validateComponents(components []Component) error {
	for i, component := range components {
		if isNilComponent(component) {
			return fmt.Errorf(
				"%w at index %d",
				ErrNilComponent,
				i,
			)
		}
	}

	return nil
}

func isNilComponent(component Component) bool {
	if component == nil {
		return true
	}

	value := reflect.ValueOf(component)

	switch value.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return value.IsNil()

	default:
		return false
	}
}
