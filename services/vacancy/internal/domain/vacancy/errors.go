package vacancy

import "errors"

var (
	ErrNotFound          = errors.New("vacancy not found")
	ErrConflict          = errors.New("vacancy conflict")
	ErrInvalidVacancy    = errors.New("invalid vacancy")
	ErrInvalidDetails    = errors.New("invalid vacancy details")
	ErrNotPublishable    = errors.New("vacancy is not ready for publication")
	ErrInvalidTransition = errors.New("operation is not allowed in the current vacancy status")
	ErrVersionConflict   = errors.New("vacancy version conflict")
)
