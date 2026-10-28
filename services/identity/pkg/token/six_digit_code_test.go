package token

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSixDigitCodeGenerate(t *testing.T) {
	t.Parallel()

	generator := NewSixDigitCode()

	for range 100 {
		code, err := generator.Generate()
		require.NoError(t, err)
		require.Regexp(t, `^[0-9]{6}$`, code)
	}
}

func TestSixDigitCodePreservesLeadingZeros(t *testing.T) {
	t.Parallel()

	generator, err := newSixDigitCode(bytes.NewReader(make([]byte, verificationCodeDigits)))
	require.NoError(t, err)

	code, err := generator.Generate()

	require.NoError(t, err)
	require.Equal(t, "000000", code)
}

func TestSixDigitCodeReturnsRandomSourceError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("random failed")
	generator, err := newSixDigitCode(errorReader{err: expectedErr})
	require.NoError(t, err)

	_, err = generator.Generate()

	require.ErrorIs(t, err, expectedErr)
}

func TestSixDigitCodeRejectsNilRandomSource(t *testing.T) {
	t.Parallel()

	generator, err := newSixDigitCode(nil)

	require.Nil(t, generator)
	require.ErrorContains(t, err, "nil random source")
}
