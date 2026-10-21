package application

import "errors"

var (
	ErrNotFound               = errors.New("job application not found")
	ErrEmptyID                = errors.New("application ID must not be empty")
	ErrEmptyVacancyID         = errors.New("vacancy ID must not be empty")
	ErrEmptyCandidateID       = errors.New("candidate ID must not be empty")
	ErrEmptyActorID           = errors.New("actor ID must not be empty")
	ErrInvalidStageTransition = errors.New("invalid stage transition")
	ErrTerminalStage          = errors.New("application is in terminal stage")
	ErrInvalidStage           = errors.New("invalid pipeline stage")
)
