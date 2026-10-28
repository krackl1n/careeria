package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Random struct {
	entropyBytes uint32
	random       io.Reader
}

func NewRandom(params Params) (*Random, error) {
	return newRandom(params, rand.Reader)
}

func newRandom(params Params, random io.Reader) (*Random, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	if random == nil {
		return nil, fmt.Errorf(
			"%w: nil random source",
			ErrInvalidParams,
		)
	}

	return &Random{
		entropyBytes: params.EntropyBytes,
		random:       random,
	}, nil
}

func (g *Random) Generate() (string, error) {
	buffer := make([]byte, g.entropyBytes)

	if _, err := io.ReadFull(g.random, buffer); err != nil {
		return "", fmt.Errorf(
			"token: generate random bytes: %w",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

var _ Generator = (*Random)(nil)
