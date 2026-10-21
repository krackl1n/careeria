package token

import "fmt"

const (
	defaultEntropyBytes uint32 = 32

	minEntropyBytes uint32 = 16
	maxEntropyBytes uint32 = 128
)

type Params struct {
	EntropyBytes uint32
}

func DefaultParams() Params {
	return Params{
		EntropyBytes: defaultEntropyBytes,
	}
}

func (p Params) validate() error {
	if p.EntropyBytes < minEntropyBytes || p.EntropyBytes > maxEntropyBytes {
		return fmt.Errorf(
			"%w: entropy bytes must be between %d and %d",
			ErrInvalidParams,
			minEntropyBytes,
			maxEntropyBytes,
		)
	}

	return nil
}
