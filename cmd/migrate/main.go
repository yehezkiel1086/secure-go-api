package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres"
)

func handleError(msg string, err error) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(1)
	}
}

func main() {
	slog.Info("running database migration...")

	conf, err := config.New()
	handleError("failed to load configuration", err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, conf.DB)
	handleError("failed to connect to postgres", err)
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		handleError("failed to execute migrations", err)
	}

	slog.Info("migration process finished")
}
