package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMinimumTime(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)
	require.Equal(t, first, MinimumTime(first, second))
	require.Equal(t, first, MinimumTime(second, first))
}

func TestNormalizeOptionalText(t *testing.T) {
	t.Parallel()

	value := "  text  "
	normalized := NormalizeOptionalText(&value)
	require.NotNil(t, normalized)
	require.Equal(t, "text", *normalized)
	require.Equal(t, "  text  ", value)

	empty := "   "
	require.Nil(t, NormalizeOptionalText(&empty))
	require.Nil(t, NormalizeOptionalText(nil))
}

func TestNormalizeRequiredText(t *testing.T) {
	t.Parallel()

	normalized, ok := NormalizeRequiredText("  text  ")
	require.True(t, ok)
	require.Equal(t, "text", normalized)

	_, ok = NormalizeRequiredText("   ")
	require.False(t, ok)
}

func TestNormalizeToken(t *testing.T) {
	t.Parallel()

	normalized, ok := NormalizeToken("  token  ")
	require.True(t, ok)
	require.Equal(t, "token", normalized)

	_, ok = NormalizeToken("two tokens")
	require.False(t, ok)
}

func TestHasText(t *testing.T) {
	t.Parallel()

	require.True(t, HasText(" text "))
	require.False(t, HasText("   "))
}

func TestEqualOptionalText(t *testing.T) {
	t.Parallel()

	left := "text"
	right := "text"
	other := "other"
	require.True(t, EqualOptionalText(nil, nil))
	require.True(t, EqualOptionalText(&left, &right))
	require.False(t, EqualOptionalText(&left, nil))
	require.False(t, EqualOptionalText(&left, &other))
}

func TestHasDuplicates(t *testing.T) {
	t.Parallel()

	require.True(t, HasDuplicates([]string{"one", "two", "one"}))
	require.False(t, HasDuplicates([]string{"one", "two"}))
}
