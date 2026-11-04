package redis

import "errors"

var ErrClosed = errors.New("redis client is closed")
