package authorizationtransaction

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewState(t *testing.T) {
	t.Parallel()

	state, err := NewState(strings.Repeat("a", maxStateLength))
	require.NoError(t, err)
	require.Len(t, state, maxStateLength)

	_, err = NewState(strings.Repeat("a", maxStateLength+1))
	require.EqualError(t, err, "OAuth state exceeds 2048 bytes")
}
