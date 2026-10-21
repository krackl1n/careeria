package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    Command
		wantErr string
	}{
		{
			name:  "up",
			value: "up",
			want:  CommandUp,
		},
		{
			name:  "status",
			value: "status",
			want:  CommandStatus,
		},
		{
			name:  "version",
			value: "version",
			want:  CommandVersion,
		},
		{
			name:    "empty",
			value:   "",
			wantErr: `unsupported migration command: ""`,
		},
		{
			name:    "down is not exposed",
			value:   "down",
			wantErr: `unsupported migration command: "down"`,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCommand(tt.value)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Empty(t, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommandValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command Command
		wantErr string
	}{
		{
			name:    "up",
			command: CommandUp,
		},
		{
			name:    "status",
			command: CommandStatus,
		},
		{
			name:    "version",
			command: CommandVersion,
		},
		{
			name:    "empty",
			command: "",
			wantErr: `unsupported migration command: ""`,
		},
		{
			name:    "unsupported",
			command: "reset",
			wantErr: `unsupported migration command: "reset"`,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.command.Validate()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
