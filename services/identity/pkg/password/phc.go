package password

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	argon2idAlgorithm = "argon2id"
	argon2idVersion   = 19

	maxEncodedHashLength = 1024
)

type encodedHash struct {
	params Params
	salt   []byte
	key    []byte
}

func encodeHash(params Params, salt, key []byte) string {
	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idAlgorithm,
		argon2idVersion,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func decodeHash(value string) (encodedHash, error) {
	if value == "" || len(value) > maxEncodedHashLength {
		return encodedHash{}, ErrInvalidHash
	}

	parts := strings.Split(value, "$")
	if len(parts) != 6 || parts[0] != "" {
		return encodedHash{}, ErrInvalidHash
	}

	if parts[1] != argon2idAlgorithm {
		return encodedHash{}, ErrInvalidHash
	}

	if err := validateVersion(parts[2]); err != nil {
		return encodedHash{}, err
	}

	params, err := parseParams(parts[3])
	if err != nil {
		return encodedHash{}, err
	}

	salt, err := decodeBase64(parts[4])
	if err != nil {
		return encodedHash{}, fmt.Errorf(
			"%w: invalid salt encoding",
			ErrInvalidHash,
		)
	}

	key, err := decodeBase64(parts[5])
	if err != nil {
		return encodedHash{}, fmt.Errorf(
			"%w: invalid key encoding",
			ErrInvalidHash,
		)
	}

	if len(salt) < int(minSaltLength) || len(salt) > int(maxSaltLength) {
		return encodedHash{}, fmt.Errorf(
			"%w: salt length outside supported range",
			ErrInvalidHash,
		)
	}

	if len(key) < int(minKeyLength) || len(key) > int(maxKeyLength) {
		return encodedHash{}, fmt.Errorf(
			"%w: key length outside supported range",
			ErrInvalidHash,
		)
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))

	return encodedHash{
		params: params,
		salt:   salt,
		key:    key,
	}, nil
}

func validateVersion(value string) error {
	raw, ok := strings.CutPrefix(value, "v=")
	if !ok || raw == "" {
		return ErrInvalidHash
	}

	version, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return fmt.Errorf(
			"%w: invalid argon2 version",
			ErrInvalidHash,
		)
	}

	if version != argon2idVersion {
		return fmt.Errorf(
			"%w: got %d, want %d",
			ErrUnsupportedVersion,
			version,
			argon2idVersion,
		)
	}

	return nil
}

func parseParams(value string) (Params, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 3 {
		return Params{}, ErrInvalidHash
	}

	memory, err := parseUint32Param(parts[0], "m")
	if err != nil {
		return Params{}, err
	}

	iterations, err := parseUint32Param(parts[1], "t")
	if err != nil {
		return Params{}, err
	}

	parallelism, err := parseUint8Param(parts[2], "p")
	if err != nil {
		return Params{}, err
	}

	if memory < minMemory || memory > maxMemory {
		return Params{}, fmt.Errorf(
			"%w: memory outside supported range",
			ErrInvalidHash,
		)
	}

	if iterations < minIterations || iterations > maxIterations {
		return Params{}, fmt.Errorf(
			"%w: iterations outside supported range",
			ErrInvalidHash,
		)
	}

	if parallelism < minParallelism || parallelism > maxParallelism {
		return Params{}, fmt.Errorf(
			"%w: parallelism outside supported range",
			ErrInvalidHash,
		)
	}

	return Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
	}, nil
}

func parseUint32Param(value, name string) (uint32, error) {
	raw, ok := strings.CutPrefix(value, name+"=")
	if !ok || raw == "" {
		return 0, ErrInvalidHash
	}

	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, ErrInvalidHash
	}

	return uint32(parsed), nil
}

func parseUint8Param(value, name string) (uint8, error) {
	raw, ok := strings.CutPrefix(value, name+"=")
	if !ok || raw == "" {
		return 0, ErrInvalidHash
	}

	parsed, err := strconv.ParseUint(raw, 10, 8)
	if err != nil {
		return 0, ErrInvalidHash
	}

	return uint8(parsed), nil
}

func decodeBase64(value string) ([]byte, error) {
	if value == "" {
		return nil, ErrInvalidHash
	}

	decoded, err := base64.RawStdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, ErrInvalidHash
	}

	return decoded, nil
}
