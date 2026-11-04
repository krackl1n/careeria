package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/krackl1n/careeria/services/identity/internal/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		slog.Error("app run", slog.Any("error", err))
		os.Exit(1)
	}
}
