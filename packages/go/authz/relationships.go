package authz

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Subject struct {
	Type     string
	ID       string
	Relation string
}

type Object struct {
	Type string
	ID   string
}

type Condition struct {
	Name    string
	Context map[string]any
}

type Tuple struct {
	Subject   Subject
	Relation  string
	Object    Object
	Condition *Condition
}

func User(subjectType, id string) Subject {
	return Subject{Type: subjectType, ID: id}
}

func Userset(subjectType, id, relation string) Subject {
	return Subject{Type: subjectType, ID: id, Relation: relation}
}

func Resource(objectType, id string) Object {
	return Object{Type: objectType, ID: id}
}

func ParseObject(value string) (Object, error) {
	objectType, id, found := strings.Cut(strings.TrimSpace(value), ":")
	if !found {
		return Object{}, errors.New("authz object must have type:id format")
	}

	object := Resource(objectType, id)
	if err := object.Validate(); err != nil {
		return Object{}, err
	}
	return object, nil
}

func (s Subject) Validate() error {
	if err := validateType(s.Type, "subject type"); err != nil {
		return err
	}
	if err := validateID(s.ID, "subject id"); err != nil {
		return err
	}
	if s.Relation != "" {
		if err := validateRelation(s.Relation); err != nil {
			return fmt.Errorf("authz subject relation is invalid: %w", err)
		}
	}
	return nil
}

func (s Subject) String() string {
	value := strings.TrimSpace(s.Type) + ":" + strings.TrimSpace(s.ID)
	if relation := strings.TrimSpace(s.Relation); relation != "" {
		value += "#" + relation
	}
	return value
}

func (o Object) Validate() error {
	if err := validateType(o.Type, "object type"); err != nil {
		return err
	}
	return validateID(o.ID, "object id")
}

func (o Object) String() string {
	return strings.TrimSpace(o.Type) + ":" + strings.TrimSpace(o.ID)
}

func (c Condition) Validate() error {
	if err := validateRelation(c.Name); err != nil {
		return fmt.Errorf("authz condition name is invalid: %w", err)
	}
	return nil
}

func (t Tuple) Validate() error {
	if err := t.Subject.Validate(); err != nil {
		return err
	}
	if err := validateRelation(t.Relation); err != nil {
		return err
	}
	if err := t.Object.Validate(); err != nil {
		return err
	}
	if t.Condition != nil {
		return t.Condition.Validate()
	}
	return nil
}

type WriteRequest struct {
	Writes                []Tuple
	Deletes               []Tuple
	IgnoreDuplicateWrites bool
	IgnoreMissingDeletes  bool
}

func (r WriteRequest) Validate() error {
	if len(r.Writes) == 0 && len(r.Deletes) == 0 {
		return errors.New("authz relationship writes or deletes are required")
	}
	for index, tuple := range r.Writes {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("authz relationship write %d is invalid: %w", index, err)
		}
	}
	for index, tuple := range r.Deletes {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("authz relationship delete %d is invalid: %w", index, err)
		}
		if tuple.Condition != nil {
			return fmt.Errorf("authz relationship delete %d must not contain a condition", index)
		}
	}
	return nil
}

type ListObjectsRequest struct {
	Subject          Subject
	Tenant           Tenant
	Relation         string
	ObjectType       string
	ConditionContext map[string]any
	ContextualTuples []Tuple
	Consistency      Consistency
}

type ListUsersRequest struct {
	Object           Object
	Tenant           Tenant
	Relation         string
	SubjectType      string
	ConditionContext map[string]any
	ContextualTuples []Tuple
	Consistency      Consistency
}

func (r ListUsersRequest) Validate() error {
	if err := r.Object.Validate(); err != nil {
		return err
	}
	if err := TenantObject(r.Tenant.ID).Validate(); err != nil {
		return err
	}
	if err := validateRelation(r.Relation); err != nil {
		return err
	}
	if err := validateType(r.SubjectType, "subject type"); err != nil {
		return err
	}
	if err := r.Consistency.Validate(); err != nil {
		return err
	}
	for index, tuple := range r.ContextualTuples {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("authz contextual tuple %d is invalid: %w", index, err)
		}
	}
	return nil
}

func (r ListObjectsRequest) Validate() error {
	if err := r.Subject.Validate(); err != nil {
		return err
	}
	if err := TenantObject(r.Tenant.ID).Validate(); err != nil {
		return err
	}
	if err := validateRelation(r.Relation); err != nil {
		return err
	}
	if err := validateType(r.ObjectType, "object type"); err != nil {
		return err
	}
	if err := r.Consistency.Validate(); err != nil {
		return err
	}
	for index, tuple := range r.ContextualTuples {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("authz contextual tuple %d is invalid: %w", index, err)
		}
	}
	return nil
}

func validateType(value, name string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("authz %s is required", name)
	}
	if strings.ContainsAny(value, ":#") || containsSpace(value) {
		return fmt.Errorf("authz %s is invalid", name)
	}
	return nil
}

func validateID(value, name string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("authz %s is required", name)
	}
	if strings.ContainsRune(value, '#') || containsSpace(value) {
		return fmt.Errorf("authz %s is invalid", name)
	}
	return nil
}

func validateRelation(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("authz relation is required")
	}
	if strings.ContainsAny(value, ":#") || containsSpace(value) {
		return errors.New("authz relation is invalid")
	}
	return nil
}

func containsSpace(value string) bool {
	return strings.IndexFunc(value, unicode.IsSpace) >= 0
}
