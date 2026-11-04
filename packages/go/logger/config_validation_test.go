package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "valid",
			cfg: Config{
				Level:   "info",
				Format:  FormatJSON,
				Service: "api-gateway",
			},
		},
		{
			name: "invalid level",
			cfg: Config{
				Level:   "verbose",
				Format:  FormatJSON,
				Service: "api-gateway",
			},
			want: "logger level",
		},
		{
			name: "missing service",
			cfg: Config{
				Level:  "info",
				Format: FormatJSON,
			},
			want: "logger service",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.want == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.want)
		})
	}
}
