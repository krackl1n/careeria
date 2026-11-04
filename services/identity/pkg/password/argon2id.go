package password

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

type Argon2id struct {
	params Params
	random io.Reader
}

func NewArgon2id(params Params) (*Argon2id, error) {
	return newArgon2id(params, rand.Reader)
}

func newArgon2id(params Params, random io.Reader) (*Argon2id, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	if random == nil {
		return nil, fmt.Errorf(
			"%w: nil random source",
			ErrInvalidParams,
		)
	}

	return &Argon2id{
		params: params,
		random: random,
	}, nil
}

func (a *Argon2id) Hash(password string) (string, error) {
	salt := make([]byte, a.params.SaltLength)

	if _, err := io.ReadFull(a.random, salt); err != nil {
		return "", fmt.Errorf(
			"password: generate salt: %w",
			err,
		)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		a.params.Iterations,
		a.params.Memory,
		a.params.Parallelism,
		a.params.KeyLength,
	)

	return encodeHash(a.params, salt, key), nil
}

func (a *Argon2id) Verify(
	password string,
	encodedHash string,
) (bool, error) {
	decoded, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	actual := argon2.IDKey(
		[]byte(password),
		decoded.salt,
		decoded.params.Iterations,
		decoded.params.Memory,
		decoded.params.Parallelism,
		decoded.params.KeyLength,
	)

	return subtle.ConstantTimeCompare(actual, decoded.key) == 1, nil
}

var (
	_ Hasher   = (*Argon2id)(nil)
	_ Verifier = (*Argon2id)(nil)
)
