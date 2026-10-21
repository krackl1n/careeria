package runtime

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Group struct {
	results <-chan Result
	done    <-chan struct{}
}

func Start(components ...Component) (*Group, error) {
	if err := validateComponents(components); err != nil {
		return nil, err
	}

	results := make(chan Result, len(components))
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(len(components))

	for _, component := range components {
		component := component

		go func() {
			defer wg.Done()

			results <- Result{
				Name: component.Name(),
				Err:  component.Start(),
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
		close(done)
	}()

	return &Group{
		results: results,
		done:    done,
	}, nil
}

func (g *Group) Results() <-chan Result {
	if g == nil {
		return nil
	}

	return g.results
}

func (g *Group) Done() <-chan struct{} {
	if g == nil {
		return nil
	}

	return g.done
}

func (g *Group) Wait(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	if g == nil {
		return nil
	}

	select {
	case <-g.done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func validateComponents(components []Component) error {
	if len(components) == 0 {
		return ErrNoComponents
	}

	names := make(map[string]struct{}, len(components))

	for i, component := range components {
		if isNilComponent(component) {
			return fmt.Errorf("%w at index %d", ErrNilComponent, i)
		}

		name := strings.TrimSpace(component.Name())
		if name == "" {
			return fmt.Errorf("%w at index %d", ErrEmptyComponentName, i)
		}

		if _, exists := names[name]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateComponentName, name)
		}

		names[name] = struct{}{}
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
