package outbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid",
			cfg: Config{
				BatchSize:      10,
				PollInterval:   time.Second,
				PublishTimeout: time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero batch size",
			cfg: Config{
				BatchSize:      0,
				PollInterval:   time.Second,
				PublishTimeout: time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative batch size",
			cfg: Config{
				BatchSize:      -1,
				PollInterval:   time.Second,
				PublishTimeout: time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero poll interval",
			cfg: Config{
				BatchSize:      10,
				PollInterval:   0,
				PublishTimeout: time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative poll interval",
			cfg: Config{
				BatchSize:      10,
				PollInterval:   -time.Second,
				PublishTimeout: time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
