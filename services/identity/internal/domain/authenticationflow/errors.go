package authenticationflow

import "errors"

var (
	ErrFactsNotFound = errors.New("authentication flow facts not found")
	ErrFactsRequired = errors.New("authentication flow facts are required")
	ErrNoNextStep    = errors.New("authentication flow has no valid next step")
)
