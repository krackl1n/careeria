package db

import (
	"embed"
	"io/fs"
)

//go:embed all:migrations
var migrationsFS embed.FS

func Migrations() fs.FS {
	return migrationsFS
}
