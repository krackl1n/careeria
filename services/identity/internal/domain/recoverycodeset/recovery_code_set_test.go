package recoverycodeset

import (
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestNewRecoveryCodeSet(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	expiresAt := now.Add(30 * 24 * time.Hour)
	definitions := recoveryCodeDefinitions(3)
	entity, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		definitions,
		&expiresAt,
		now,
	)
	require.NoError(t, err)
	require.Len(t, entity.Codes, 3)
	require.Equal(t, definitions[0].ID, entity.Codes[0].ID)
	require.Equal(t, definitions[0].Hash, entity.Codes[0].Hash)
	require.Equal(t, now, entity.Codes[0].CreatedAt)
	require.Equal(t, expiresAt, *entity.ExpiresAt)
}

func TestRecoveryCodeCanOnlyBeUsedOnce(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	entity := newRecoveryCodeSetForTest(t, 2, nil, now)
	codeID := entity.Codes[0].ID

	outcome, err := entity.UseCode(codeID, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, UseOutcomeUsed, outcome)
	require.Equal(t, now.Add(time.Minute), *entity.Codes[0].UsedAt)
	require.Nil(t, entity.DisabledAt)

	_, err = entity.UseCode(codeID, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrCodeUsed)
	require.Equal(t, now.Add(time.Minute), *entity.Codes[0].UsedAt)
}

func TestLastRecoveryCodeExhaustsAndDisablesSet(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	entity := newRecoveryCodeSetForTest(t, 1, nil, now)
	usedAt := now.Add(time.Minute)

	outcome, err := entity.UseCode(entity.Codes[0].ID, usedAt)
	require.NoError(t, err)
	require.Equal(t, UseOutcomeExhausted, outcome)
	require.Equal(t, usedAt, *entity.Codes[0].UsedAt)
	require.Equal(t, usedAt, *entity.DisabledAt)

	_, err = entity.UseCode(entity.Codes[0].ID, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrDisabled)
}

func TestExpiredRecoveryCodeSetCannotBeUsed(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	expiresAt := now.Add(time.Hour)
	entity := newRecoveryCodeSetForTest(t, 2, &expiresAt, now)

	_, err := entity.UseCode(entity.Codes[0].ID, expiresAt)
	require.ErrorIs(t, err, ErrExpired)
	require.Nil(t, entity.Codes[0].UsedAt)
}

func TestDisableRecoveryCodeSetPreservesUnusedCodes(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	entity := newRecoveryCodeSetForTest(t, 2, nil, now)
	disabledAt := now.Add(time.Minute)

	require.NoError(t, entity.Disable(disabledAt))
	require.Equal(t, disabledAt, *entity.DisabledAt)
	require.Nil(t, entity.Codes[0].UsedAt)
	require.NoError(t, entity.Disable(now.Add(2*time.Minute)))
	require.Equal(t, disabledAt, *entity.DisabledAt)
}

func TestNewRecoveryCodeSetRejectsDuplicateCodes(t *testing.T) {
	t.Parallel()

	now := recoveryCodeTestTime()
	codeID := CodeID(uuid.New())
	_, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		[]CodeDefinition{
			{ID: codeID, Hash: CodeHash("hash-one")},
			{ID: codeID, Hash: CodeHash("hash-two")},
		},
		nil,
		now,
	)
	require.Error(t, err)
}

func newRecoveryCodeSetForTest(t *testing.T, count int, expiresAt *time.Time, now time.Time) *RecoveryCodeSet {
	t.Helper()

	entity, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		recoveryCodeDefinitions(count),
		expiresAt,
		now,
	)
	require.NoError(t, err)
	return entity
}

func recoveryCodeDefinitions(count int) []CodeDefinition {
	definitions := make([]CodeDefinition, 0, count)
	for index := range count {
		definitions = append(definitions, CodeDefinition{
			ID:   CodeID(uuid.New()),
			Hash: CodeHash("hash-" + string(rune('a'+index))),
		})
	}
	return definitions
}

func recoveryCodeTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
