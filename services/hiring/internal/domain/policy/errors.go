package policy

import "errors"

var (
	ErrNilApplication            = errors.New("application is required")
	ErrInvalidStageForOffer      = errors.New("application must be in INTERVIEW stage to initiate offer")
	ErrMissingCompletedInterview = errors.New("at least one interview must be completed with feedback")
)
