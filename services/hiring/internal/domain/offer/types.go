package offer

import (
	"fmt"
	"strings"
)

type OfferStatus string

const (
	StatusDraft    OfferStatus = "DRAFT"
	StatusSent     OfferStatus = "SENT"
	StatusAccepted OfferStatus = "ACCEPTED"
	StatusDeclined OfferStatus = "DECLINED"
	StatusExpired  OfferStatus = "EXPIRED"
	StatusRevoked  OfferStatus = "REVOKED"
)

func (s OfferStatus) String() string {
	return string(s)
}

func (s OfferStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusSent, StatusAccepted, StatusDeclined, StatusExpired, StatusRevoked:
		return true
	default:
		return false
	}
}

func (s OfferStatus) IsTerminal() bool {
	switch s {
	case StatusAccepted, StatusDeclined, StatusExpired, StatusRevoked:
		return true
	default:
		return false
	}
}

type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyCNY Currency = "CNY"
	CurrencyKZT Currency = "KZT"
	CurrencyBYN Currency = "BYN"
)

func (c Currency) String() string {
	return string(c)
}

func (c Currency) Valid() bool {
	if len(c) != 3 {
		return false
	}
	for _, ch := range c {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func ParseCurrency(value string) (Currency, error) {
	c := Currency(strings.ToUpper(strings.TrimSpace(value)))
	if !c.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, value)
	}
	return c, nil
}
