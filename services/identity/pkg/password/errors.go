package password

import "errors"

var (
	ErrInvalidParams      = errors.New("password: invalid parameters")
	ErrInvalidHash        = errors.New("password: invalid hash")
	ErrUnsupportedVersion = errors.New("password: unsupported argon2 version")
)
