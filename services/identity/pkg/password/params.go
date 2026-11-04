package password

import "fmt"

const (
	defaultMemory      uint32 = 64 * 1024
	defaultIterations  uint32 = 3
	defaultParallelism uint8  = 2
	defaultSaltLength  uint32 = 16
	defaultKeyLength   uint32 = 32

	minMemory      uint32 = 8 * 1024
	maxMemory      uint32 = 256 * 1024
	minIterations  uint32 = 1
	maxIterations  uint32 = 10
	minParallelism uint8  = 1
	maxParallelism uint8  = 16
	minSaltLength  uint32 = 16
	maxSaltLength  uint32 = 64
	minKeyLength   uint32 = 16
	maxKeyLength   uint32 = 64
)

type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Params {
	return Params{
		Memory:      defaultMemory,
		Iterations:  defaultIterations,
		Parallelism: defaultParallelism,
		SaltLength:  defaultSaltLength,
		KeyLength:   defaultKeyLength,
	}
}

func (p Params) validate() error {
	if p.Memory < minMemory || p.Memory > maxMemory {
		return fmt.Errorf(
			"%w: memory must be between %d and %d KiB",
			ErrInvalidParams,
			minMemory,
			maxMemory,
		)
	}

	if p.Iterations < minIterations || p.Iterations > maxIterations {
		return fmt.Errorf(
			"%w: iterations must be between %d and %d",
			ErrInvalidParams,
			minIterations,
			maxIterations,
		)
	}

	if p.Parallelism < minParallelism || p.Parallelism > maxParallelism {
		return fmt.Errorf(
			"%w: parallelism must be between %d and %d",
			ErrInvalidParams,
			minParallelism,
			maxParallelism,
		)
	}

	if p.SaltLength < minSaltLength || p.SaltLength > maxSaltLength {
		return fmt.Errorf(
			"%w: salt length must be between %d and %d bytes",
			ErrInvalidParams,
			minSaltLength,
			maxSaltLength,
		)
	}

	if p.KeyLength < minKeyLength || p.KeyLength > maxKeyLength {
		return fmt.Errorf(
			"%w: key length must be between %d and %d bytes",
			ErrInvalidParams,
			minKeyLength,
			maxKeyLength,
		)
	}

	return nil
}
