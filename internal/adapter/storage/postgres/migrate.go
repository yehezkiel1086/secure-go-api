package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yehezkiel1086/secure-go-api/db/migrations"
)

// Migrate ensures the tracking table exists and runs any unapplied embedded migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// 1. Create schema_migrations table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := pool.Exec(ctx, createTableQuery); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// 2. Check if migration has already been applied
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1);`
	if err := pool.QueryRow(ctx, checkQuery, migrations.Version).Scan(&exists); err != nil {
		return fmt.Errorf("checking migration status: %w", err)
	}

	if exists {
		slog.Info("database schema is up to date", "version", migrations.Version)
		return nil
	}

	// 3. Apply migration inside a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting migration transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	slog.Info("applying database migration", "version", migrations.Version)
	if _, err := tx.Exec(ctx, migrations.SchemaSQL); err != nil {
		return fmt.Errorf("executing schema.sql: %w", err)
	}

	// 4. Record applied migration
	recordQuery := `INSERT INTO schema_migrations (version) VALUES ($1);`
	if _, err := tx.Exec(ctx, recordQuery, migrations.Version); err != nil {
		return fmt.Errorf("recording applied migration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}

	slog.Info("database migration applied successfully", "version", migrations.Version)
	return nil
}
