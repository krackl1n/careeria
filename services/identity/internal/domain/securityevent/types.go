package securityevent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	serviceaccountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
)

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type EventType string

func NewEventType(value string) (EventType, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("security event type must be a non-empty token")
	}

	return EventType(value), nil
}

type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
	ResultBlocked Result = "blocked"
)

func ParseResult(value string) (Result, error) {
	result := Result(value)
	switch result {
	case ResultSuccess, ResultFailure, ResultBlocked:
		return result, nil
	default:
		return "", errors.New("unknown security event result")
	}
}

type ActorType string

const (
	ActorTypeAnonymous      ActorType = "anonymous"
	ActorTypeIdentity       ActorType = "identity"
	ActorTypeServiceAccount ActorType = "service_account"
	ActorTypeAdministrator  ActorType = "administrator"
	ActorTypeSystem         ActorType = "system"
)

func ParseActorType(value string) (ActorType, error) {
	actorType := ActorType(value)
	switch actorType {
	case ActorTypeAnonymous, ActorTypeIdentity, ActorTypeServiceAccount, ActorTypeAdministrator, ActorTypeSystem:
		return actorType, nil
	default:
		return "", errors.New("unknown security actor type")
	}
}

type Actor struct {
	Type ActorType

	IdentityID       *identitydomain.ID
	ServiceAccountID *serviceaccountdomain.ID
}

func NewAnonymousActor() Actor {
	return Actor{Type: ActorTypeAnonymous}
}

func NewIdentityActor(identityID identitydomain.ID) (Actor, error) {
	return newIdentityActor(ActorTypeIdentity, identityID)
}

func NewAdministratorActor(identityID identitydomain.ID) (Actor, error) {
	return newIdentityActor(ActorTypeAdministrator, identityID)
}

func NewServiceAccountActor(serviceAccountID serviceaccountdomain.ID) (Actor, error) {
	if serviceAccountID.UUID() == uuid.Nil {
		return Actor{}, errors.New("create service account security actor: identifier is invalid")
	}

	return Actor{
		Type:             ActorTypeServiceAccount,
		ServiceAccountID: domain.ClonePointer(&serviceAccountID),
	}, nil
}

func NewSystemActor() Actor {
	return Actor{Type: ActorTypeSystem}
}

func newIdentityActor(actorType ActorType, identityID identitydomain.ID) (Actor, error) {
	if !identityID.IsValid() {
		return Actor{}, fmt.Errorf("create %s security actor: identifier is invalid", actorType)
	}

	return Actor{
		Type:       actorType,
		IdentityID: domain.ClonePointer(&identityID),
	}, nil
}

func copyActor(a Actor) Actor {
	a.IdentityID = domain.ClonePointer(a.IdentityID)
	a.ServiceAccountID = domain.ClonePointer(a.ServiceAccountID)
	return a
}

type IdentifierHash []byte

func NewIdentifierHash(value []byte) (IdentifierHash, error) {
	if len(value) == 0 {
		return nil, errors.New("security event identifier hash is required")
	}

	return IdentifierHash(slices.Clone(value)), nil
}

type Metadata []byte

func NewMetadata(value []byte) (Metadata, error) {
	if len(bytes.TrimSpace(value)) == 0 {
		value = []byte(`{}`)
	}
	if err := validateMetadata(value); err != nil {
		return nil, fmt.Errorf("create security event metadata: %w", err)
	}

	return Metadata(slices.Clone(value)), nil
}

func validateMetadata(value []byte) error {
	value = bytes.TrimSpace(value)
	if len(value) == 0 || value[0] != '{' {
		return errors.New("security event metadata must be a JSON object")
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil {
		return fmt.Errorf("decode security event metadata: %w", err)
	}
	if object == nil {
		return errors.New("security event metadata must be a JSON object")
	}

	return nil
}
