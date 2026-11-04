package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeHash(t *testing.T) {
	params := DefaultParams()

	salt := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c,
		0x0d, 0x0e, 0x0f, 0x10,
	}

	key := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c,
		0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14,
		0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c,
		0x1d, 0x1e, 0x1f, 0x20,
	}

	encoded := encodeHash(params, salt, key)

	decoded, err := decodeHash(encoded)
	require.NoError(t, err)

	assert.Equal(t, params, decoded.params)
	assert.Equal(t, salt, decoded.salt)
	assert.Equal(t, key, decoded.key)
}

func TestDecodeHash(t *testing.T) {
	const valid = "$argon2id$v=19$m=65536,t=3,p=2$" +
		"AQIDBAUGBwgJCgsMDQ4PEA$" +
		"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA"

	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{
			name:  "valid",
			value: valid,
		},
		{
			name:    "empty",
			value:   "",
			wantErr: ErrInvalidHash,
		},
		{
			name: "missing leading delimiter",
			value: "argon2id$v=19$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "wrong algorithm",
			value: "$argon2i$v=19$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "wrong version",
			value: "$argon2id$v=18$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrUnsupportedVersion,
		},
		{
			name: "missing parameter",
			value: "$argon2id$v=19$m=65536,t=3$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "invalid parameter order",
			value: "$argon2id$v=19$t=3,m=65536,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "memory too large",
			value: "$argon2id$v=19$m=999999999,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "iterations too large",
			value: "$argon2id$v=19$m=65536,t=100,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "parallelism too large",
			value: "$argon2id$v=19$m=65536,t=3,p=100$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "invalid salt base64",
			value: "$argon2id$v=19$m=65536,t=3,p=2$%%%$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "short salt",
			value: "$argon2id$v=19$m=65536,t=3,p=2$AQIDBA$" +
				"AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA",
			wantErr: ErrInvalidHash,
		},
		{
			name: "short key",
			value: "$argon2id$v=19$m=65536,t=3,p=2$" +
				"AQIDBAUGBwgJCgsMDQ4PEA$AQIDBA",
			wantErr: ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeHash(tt.value)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, uint32(64*1024), result.params.Memory)
			assert.Equal(t, uint32(3), result.params.Iterations)
			assert.Equal(t, uint8(2), result.params.Parallelism)
			assert.Equal(t, uint32(16), result.params.SaltLength)
			assert.Equal(t, uint32(32), result.params.KeyLength)
		})
	}
}
