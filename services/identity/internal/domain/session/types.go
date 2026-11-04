package session

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type IPAddress string

func NewIPAddress(value string) (IPAddress, error) {
	value = strings.TrimSpace(value)
	address, err := netip.ParseAddr(value)
	if err != nil {
		return "", fmt.Errorf("parse session IP address: %w", err)
	}

	return IPAddress(address.String()), nil
}
