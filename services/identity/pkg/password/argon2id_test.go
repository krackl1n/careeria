package password

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArgon2id(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
	}{
		{
			name:   "default",
			params: DefaultParams(),
		},
		{
			name: "minimum",
			params: Params{
				Memory:      minMemory,
				Iterations:  minIterations,
				Parallelism: minParallelism,
				SaltLength:  minSaltLength,
				KeyLength:   minKeyLength,
			},
		},
		{
			name: "maximum",
			params: Params{
				Memory:      maxMemory,
				Iterations:  maxIterations,
				Parallelism: maxParallelism,
				SaltLength:  maxSaltLength,
				KeyLength:   maxKeyLength,
			},
		},
		{
			name: "memory too small",
			params: Params{
				Memory:      minMemory - 1,
				Iterations:  defaultIterations,
				Parallelism: defaultParallelism,
				SaltLength:  defaultSaltLength,
				KeyLength:   defaultKeyLength,
			},
			wantErr: true,
		},
		{
			name: "memory too large",
			params: Params{
				Memory:      maxMemory + 1,
				Iterations:  defaultIterations,
				Parallelism: defaultParallelism,
				SaltLength:  defaultSaltLength,
				KeyLength:   defaultKeyLength,
			},
			wantErr: true,
		},
		{
			name: "iterations too small",
			params: Params{
				Memory:      defaultMemory,
				Iterations:  0,
				Parallelism: defaultParallelism,
				SaltLength:  defaultSaltLength,
				KeyLength:   defaultKeyLength,
			},
			wantErr: true,
		},
		{
			name: "parallelism too small",
			params: Params{
				Memory:      defaultMemory,
				Iterations:  defaultIterations,
				Parallelism: 0,
				SaltLength:  defaultSaltLength,
				KeyLength:   defaultKeyLength,
			},
			wantErr: true,
		},
		{
			name: "salt too small",
			params: Params{
				Memory:      defaultMemory,
				Iterations:  defaultIterations,
				Parallelism: defaultParallelism,
				SaltLength:  minSaltLength - 1,
				KeyLength:   defaultKeyLength,
			},
			wantErr: true,
		},
		{
			name: "key too small",
			params: Params{
				Memory:      defaultMemory,
				Iterations:  defaultIterations,
				Parallelism: defaultParallelism,
				SaltLength:  defaultSaltLength,
				KeyLength:   minKeyLength - 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher, err := NewArgon2id(tt.params)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidParams)
				assert.Nil(t, hasher)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, hasher)
		})
	}
}

func TestArgon2id_Hash(t *testing.T) {
	hasher, err := NewArgon2id(DefaultParams())
	require.NoError(t, err)

	hash, err := hasher.Hash("correct horse battery staple")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(
		hash,
		"$argon2id$v=19$m=65536,t=3,p=2$",
	))
}

func TestArgon2id_HashUsesRandomSalt(t *testing.T) {
	hasher, err := NewArgon2id(DefaultParams())
	require.NoError(t, err)

	first, err := hasher.Hash("password")
	require.NoError(t, err)

	second, err := hasher.Hash("password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestArgon2id_Verify(t *testing.T) {
	hasher, err := NewArgon2id(DefaultParams())
	require.NoError(t, err)

	hash, err := hasher.Hash("correct horse battery staple")
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: "correct horse battery staple",
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrong password",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := hasher.Verify(tt.password, hash)

			require.NoError(t, err)
			assert.Equal(t, tt.want, ok)
		})
	}
}

func TestArgon2id_EmptyPassword(t *testing.T) {
	hasher, err := NewArgon2id(DefaultParams())
	require.NoError(t, err)

	hash, err := hasher.Hash("")
	require.NoError(t, err)

	ok, err := hasher.Verify("", hash)
	require.NoError(t, err)

	assert.True(t, ok)
}

func TestArgon2id_VerifyInvalidHash(t *testing.T) {
	hasher, err := NewArgon2id(DefaultParams())
	require.NoError(t, err)

	tests := []struct {
		name    string
		hash    string
		wantErr error
	}{
		{
			name:    "empty",
			hash:    "",
			wantErr: ErrInvalidHash,
		},
		{
			name:    "garbage",
			hash:    "not-a-hash",
			wantErr: ErrInvalidHash,
		},
		{
			name: "wrong algorithm",
			hash: "$argon2i$v=19$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "unsupported version",
			hash: "$argon2id$v=18$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrUnsupportedVersion,
		},
		{
			name: "invalid memory",
			hash: "$argon2id$v=19$m=nope,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "memory too large",
			hash: "$argon2id$v=19$m=999999999,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "invalid salt",
			hash: "$argon2id$v=19$m=65536,t=3,p=2$%%%$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "salt too short",
			hash: "$argon2id$v=19$m=65536,t=3,p=2$AQIDBA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := hasher.Verify("password", tt.hash)

			assert.False(t, ok)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestArgon2id_HashRandomFailure(t *testing.T) {
	expectedErr := errors.New("random failed")

	hasher, err := newArgon2id(
		DefaultParams(),
		errorReader{err: expectedErr},
	)
	require.NoError(t, err)

	hash, err := hasher.Hash("password")

	assert.Empty(t, hash)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewArgon2id_NilRandom(t *testing.T) {
	hasher, err := newArgon2id(DefaultParams(), nil)

	assert.Nil(t, hasher)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidParams)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
