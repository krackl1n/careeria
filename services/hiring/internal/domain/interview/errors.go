package interview

import "errors"

var (
	ErrNotFound                 = errors.New("interview not found")
	ErrEmptyID                  = errors.New("interview ID must not be empty")
	ErrEmptyApplicationID       = errors.New("application ID must not be empty")
	ErrEmptyInterviewType       = errors.New("interview type must not be empty")
	ErrStartInPast              = errors.New("interview cannot be scheduled in the past")
	ErrInvalidInterval          = errors.New("interview end time must be strictly after start time")
	ErrEmptyParticipants        = errors.New("interview must have at least one participant")
	ErrInvalidParticipant       = errors.New("participant user ID must not be empty")
	ErrInvalidStatus            = errors.New("invalid interview status")
	ErrInterviewCancelled       = errors.New("cannot submit feedback for cancelled interview")
	ErrNotParticipant           = errors.New("user is not an assigned interviewer for this interview")
	ErrFeedbackAlreadySubmitted = errors.New("feedback from this interviewer already exists")
	ErrEmptyInterviewerID       = errors.New("interviewer user ID must not be empty")
	ErrInvalidScore             = errors.New("score must be between 1 and 5")
)
