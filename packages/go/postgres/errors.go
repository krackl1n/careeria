package postgres

import "errors"

var ErrClosed = errors.New("postgres client is closed")
