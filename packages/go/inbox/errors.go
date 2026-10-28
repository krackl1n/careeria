package inbox

import "errors"

var (
	ErrNilTxManager  = errors.New("inbox: tx manager is nil")
	ErrNilRepository = errors.New("inbox: repository is nil")
	ErrNilHandler    = errors.New("inbox: handler is nil")
)
