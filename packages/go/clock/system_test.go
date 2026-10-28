package clock_test

import (
	"testing"
	"time"

	"github.com/krackl1n/careeria/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemNow(t *testing.T) {
	t.Parallel()

	c := clock.New()

	before := time.Now()
	now := c.Now()
	after := time.Now()

	require.False(t, now.Before(before))
	require.False(t, now.After(after))
}

func TestSystemImplementsClock(t *testing.T) {
	t.Parallel()

	var c clock.Clock = clock.New()

	assert.NotNil(t, c)
}
