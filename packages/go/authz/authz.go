package authz

import (
	"context"
	"errors"
	"fmt"
)

var ErrDenied = errors.New("authz: access denied")

type Actor struct {
	Type string
	ID   string
}

type Tenant struct {
	ID string
}

type Access struct {
	Actor            Actor
	Tenant           Tenant
	Permission       string
	Resource         Object
	ConditionContext map[string]any
	Consistency      Consistency
}

func (a Access) Validate() error {
	if err := User(a.Actor.Type, a.Actor.ID).Validate(); err != nil {
		return err
	}
	if err := TenantObject(a.Tenant.ID).Validate(); err != nil {
		return err
	}
	if err := validateRelation(a.Permission); err != nil {
		return err
	}
	if err := a.Resource.Validate(); err != nil {
		return err
	}
	return a.Consistency.Validate()
}

func TenantObject(id string) Object {
	return Resource("tenant", id)
}

type Consistency uint8

const (
	ConsistencyDefault Consistency = iota
	ConsistencyMinimizeLatency
	ConsistencyHigherConsistency
)

func (c Consistency) Validate() error {
	switch c {
	case ConsistencyDefault, ConsistencyMinimizeLatency, ConsistencyHigherConsistency:
		return nil
	default:
		return fmt.Errorf("authz consistency %d is unsupported", c)
	}
}

type Decision struct {
	Allowed bool
}

type Authorizer interface {
	Check(context.Context, Access) (Decision, error)
	Require(context.Context, Access) error
}

type RelationshipStore interface {
	Write(context.Context, WriteRequest) error
	ListObjects(context.Context, ListObjectsRequest) ([]Object, error)
}

type DeniedError struct {
	Access Access
}

func (e *DeniedError) Error() string {
	if e == nil {
		return ErrDenied.Error()
	}

	return fmt.Sprintf(
		"%s: actor %q does not have permission %q for object %q in tenant %q",
		ErrDenied,
		User(e.Access.Actor.Type, e.Access.Actor.ID).String(),
		e.Access.Permission,
		e.Access.Resource.String(),
		TenantObject(e.Access.Tenant.ID).String(),
	)
}

func (e *DeniedError) Unwrap() error {
	return ErrDenied
}
