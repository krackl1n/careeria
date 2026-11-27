package vacancy

import (
	"fmt"
	"regexp"
	"strings"
)

type TaxBasis string

const (
	TaxGross TaxBasis = "GROSS"
	TaxNet   TaxBasis = "NET"
)

// Amount is a non-negative number of hundredths of the stated currency (NUMERIC(18,2)).
type Amount int64

const maxAmount Amount = 999_999_999_999_999_999

var currencyCode = regexp.MustCompile("^[A-Z]{3}$")

type Compensation struct {
	Minimum  *Amount
	Maximum  *Amount
	Currency string
	Period   string
	TaxBasis TaxBasis
	Bonus    *string
}

func (c *Compensation) clone() *Compensation {
	if c == nil {
		return nil
	}
	copy := *c
	copy.Minimum = clonePtr(c.Minimum)
	copy.Maximum = clonePtr(c.Maximum)
	copy.Bonus = clonePtr(c.Bonus)
	return &copy
}

func (c *Compensation) validate() error {
	if c == nil {
		return nil
	}
	for _, amount := range []*Amount{c.Minimum, c.Maximum} {
		if amount != nil && (*amount < 0 || *amount > maxAmount) {
			return fmt.Errorf("%w: compensation amount is outside NUMERIC(18,2) range", ErrInvalidDetails)
		}
	}
	if c.Minimum != nil && c.Maximum != nil && *c.Minimum > *c.Maximum {
		return fmt.Errorf("%w: minimum compensation exceeds maximum", ErrInvalidDetails)
	}
	if c.Currency != "" && !currencyCode.MatchString(c.Currency) {
		return fmt.Errorf("%w: currency must be a three-letter uppercase code", ErrInvalidDetails)
	}
	if (c.Minimum != nil || c.Maximum != nil) && (c.Currency == "" || strings.TrimSpace(c.Period) == "") {
		return fmt.Errorf("%w: compensation amount requires a currency and period", ErrInvalidDetails)
	}
	switch c.TaxBasis {
	case "", TaxGross, TaxNet:
		return nil
	default:
		return fmt.Errorf("%w: unknown tax basis %q", ErrInvalidDetails, c.TaxBasis)
	}
}
