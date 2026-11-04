package authentication

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		methods []Method
		level   Level
	}{
		{name: "password", methods: []Method{MethodPassword}, level: LevelSingleFactor},
		{name: "WebAuthn", methods: []Method{MethodWebAuthn}, level: LevelSingleFactor},
		{
			name:    "password and TOTP",
			methods: []Method{MethodPassword, MethodTOTP},
			level:   LevelMultiFactor,
		},
		{
			name:    "password and WebAuthn",
			methods: []Method{MethodPassword, MethodWebAuthn},
			level:   LevelMultiFactor,
		},
		{
			name:    "password and recovery code",
			methods: []Method{MethodPassword, MethodRecoveryCode},
			level:   LevelMultiFactor,
		},
		{
			name:    "methods from the same factor class",
			methods: []Method{MethodTOTP, MethodWebAuthn},
			level:   LevelSingleFactor,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			level, err := ResolveLevel(test.methods)
			require.NoError(t, err)
			require.Equal(t, test.level, level)
		})
	}
}

func TestResolveLevelRejectsInvalidMethods(t *testing.T) {
	t.Parallel()

	_, err := ResolveLevel(nil)
	require.Error(t, err)

	_, err = ResolveLevel([]Method{MethodPassword, MethodPassword})
	require.Error(t, err)

	_, err = ResolveLevel([]Method{"unknown"})
	require.Error(t, err)
}
