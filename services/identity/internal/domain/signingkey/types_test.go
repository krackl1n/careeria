package signingkey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAlgorithm(t *testing.T) {
	t.Parallel()

	for _, value := range []Algorithm{AlgorithmEdDSA, AlgorithmES256, AlgorithmRS256} {
		value := value
		t.Run(string(value), func(t *testing.T) {
			t.Parallel()

			actual, err := NewAlgorithm(string(value))
			require.NoError(t, err)
			require.Equal(t, value, actual)
		})
	}

	normalized, err := NewAlgorithm(" EdDSA ")
	require.NoError(t, err)
	require.Equal(t, AlgorithmEdDSA, normalized)

	for _, value := range []string{"", "HS256", "none"} {
		value := value
		t.Run("reject_"+value, func(t *testing.T) {
			t.Parallel()

			_, err := NewAlgorithm(value)
			require.Error(t, err)
		})
	}
}
