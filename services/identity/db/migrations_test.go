package db

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationsGuardSchemaObjectCreation(t *testing.T) {
	t.Parallel()

	err := fs.WalkDir(migrationsFS, "migrations", func(path string, entry fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		contents, err := fs.ReadFile(migrationsFS, path)
		require.NoError(t, err)
		for _, line := range strings.Split(string(contents), "\n") {
			if strings.HasPrefix(line, "CREATE TABLE ") {
				require.Truef(t, strings.HasPrefix(line, "CREATE TABLE IF NOT EXISTS "), "%s contains unguarded %q", path, line)
			}
			if strings.HasPrefix(line, "CREATE INDEX ") {
				require.Truef(t, strings.HasPrefix(line, "CREATE INDEX IF NOT EXISTS "), "%s contains unguarded %q", path, line)
			}
			if strings.HasPrefix(line, "CREATE UNIQUE INDEX ") {
				require.Truef(t, strings.HasPrefix(line, "CREATE UNIQUE INDEX IF NOT EXISTS "), "%s contains unguarded %q", path, line)
			}
		}
		return nil
	})
	require.NoError(t, err)
}
