package identity

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxEmailLength = 320

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

type ID uuid.UUID
type EmailID uuid.UUID
type PhoneID uuid.UUID
type ExternalAccountID uuid.UUID

func (id ID) UUID() uuid.UUID                { return uuid.UUID(id) }
func (id EmailID) UUID() uuid.UUID           { return uuid.UUID(id) }
func (id PhoneID) UUID() uuid.UUID           { return uuid.UUID(id) }
func (id ExternalAccountID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool                { return uuid.UUID(id) != uuid.Nil }
func (id EmailID) IsValid() bool           { return uuid.UUID(id) != uuid.Nil }
func (id PhoneID) IsValid() bool           { return uuid.UUID(id) != uuid.Nil }
func (id ExternalAccountID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type Status string

const (
	StatusPending  Status = "pending"
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusDeleted  Status = "deleted"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusPending, StatusActive, StatusDisabled, StatusDeleted:
		return status, nil
	default:
		return "", errors.New("unknown identity status")
	}
}

type EmailAddress string

func NewEmailAddress(value string) (EmailAddress, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", errors.New("email is required")
	}
	if utf8.RuneCountInString(value) > maxEmailLength {
		return "", errors.New("email is too long")
	}

	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || strings.Count(value, "@") != 1 {
		return "", errors.New("email is invalid")
	}

	return EmailAddress(value), nil
}

type PhoneNumber string

func NewPhoneNumber(value string) (PhoneNumber, error) {
	value = strings.TrimSpace(value)
	if !e164Pattern.MatchString(value) {
		return "", errors.New("phone must be in E.164 format")
	}

	return PhoneNumber(value), nil
}

type ExternalProvider string

func NewExternalProvider(value string) (ExternalProvider, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", errors.New("external provider is required")
	}

	return ExternalProvider(value), nil
}

type ExternalSubject string

func NewExternalSubject(value string) (ExternalSubject, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("external subject is required")
	}

	return ExternalSubject(value), nil
}
