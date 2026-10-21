package logger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantKey string
		wantNil bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantKey: "",
			wantNil: true,
		},
		{
			name:    "error",
			err:     errors.New("failed"),
			wantKey: "error",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Error(tt.err)

			require.Equal(t, tt.wantKey, got.Key)

			if tt.wantNil {
				require.Equal(t, "<nil>", got.Value.String())
				return
			}

			require.Equal(t, "failed", got.Value.String())
		})
	}
}
