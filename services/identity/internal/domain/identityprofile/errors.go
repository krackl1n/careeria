package identityprofile

import "errors"

var (
	ErrAlreadyExists     = errors.New("identity profile already exists")
	ErrNotFound          = errors.New("identity profile not found")
	ErrExternallyManaged = errors.New("identity profile is managed by an external source")
)
