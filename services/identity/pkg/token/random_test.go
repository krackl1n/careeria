package token

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandom(t *testing.T) {
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
				EntropyBytes: minEntropyBytes,
			},
		},
		{
			name: "maximum",
			params: Params{
				EntropyBytes: maxEntropyBytes,
			},
		},
		{
			name: "too small",
			params: Params{
				EntropyBytes: minEntropyBytes - 1,
			},
			wantErr: true,
		},
		{
			name: "too large",
			params: Params{
				EntropyBytes: maxEntropyBytes + 1,
			},
			wantErr: true,
		},
		{
			name:    "zero",
			params:  Params{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewRandom(tt.params)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidParams)
				assert.Nil(t, generator)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, generator)
		})
	}
}

func TestRandom_Generate(t *testing.T) {
	generator, err := NewRandom(DefaultParams())
	require.NoError(t, err)

	value, err := generator.Generate()
	require.NoError(t, err)

	assert.NotEmpty(t, value)

	decoded, err := base64.RawURLEncoding.DecodeString(value)
	require.NoError(t, err)

	assert.Len(t, decoded, int(defaultEntropyBytes))
}

func TestRandom_GenerateUniqueValues(t *testing.T) {
	generator, err := NewRandom(DefaultParams())
	require.NoError(t, err)

	first, err := generator.Generate()
	require.NoError(t, err)

	second, err := generator.Generate()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestRandom_GenerateExpectedLength(t *testing.T) {
	tests := []struct {
		name         string
		entropyBytes uint32
	}{
		{
			name:         "minimum",
			entropyBytes: minEntropyBytes,
		},
		{
			name:         "default",
			entropyBytes: defaultEntropyBytes,
		},
		{
			name:         "maximum",
			entropyBytes: maxEntropyBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewRandom(Params{
				EntropyBytes: tt.entropyBytes,
			})
			require.NoError(t, err)

			value, err := generator.Generate()
			require.NoError(t, err)

			decoded, err := base64.RawURLEncoding.DecodeString(value)
			require.NoError(t, err)

			assert.Len(t, decoded, int(tt.entropyBytes))
		})
	}
}

func TestRandom_GenerateRandomFailure(t *testing.T) {
	expectedErr := errors.New("random failed")

	generator, err := newRandom(
		DefaultParams(),
		errorReader{err: expectedErr},
	)
	require.NoError(t, err)

	value, err := generator.Generate()

	assert.Empty(t, value)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewRandom_NilRandom(t *testing.T) {
	generator, err := newRandom(DefaultParams(), nil)

	assert.Nil(t, generator)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidParams)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
