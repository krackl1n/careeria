package offer

import "errors"

var (
	ErrNotFound            = errors.New("job offer not found")
	ErrEmptyID             = errors.New("offer ID must not be empty")
	ErrEmptyApplicationID  = errors.New("application ID must not be empty")
	ErrInvalidCompensation = errors.New("compensation must be strictly positive")
	ErrInvalidCurrency     = errors.New("currency must be a 3-letter ISO code")
	ErrEmptyGrade          = errors.New("grade must not be empty")
	ErrInvalidExpiration   = errors.New("offer validity period must be at least 24 hours from creation")
	ErrInvalidStatus       = errors.New("invalid offer status")
	ErrOfferExpired        = errors.New("offer has expired")
	ErrOfferNotExpired     = errors.New("offer expiration time has not been reached yet")
)
