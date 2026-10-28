package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdvisoryLockID(t *testing.T) {
	t.Parallel()

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()

		first := advisoryLockID("identity")
		second := advisoryLockID("identity")

		assert.Equal(t, first, second)
	})

	t.Run("different service names", func(t *testing.T) {
		t.Parallel()

		identity := advisoryLockID("identity")
		payments := advisoryLockID("payments")

		assert.NotEqual(t, identity, payments)
	})

	t.Run("non negative", func(t *testing.T) {
		t.Parallel()

		lockID := advisoryLockID("identity")

		assert.GreaterOrEqual(t, lockID, int64(0))
	})
}
