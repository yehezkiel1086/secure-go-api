package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/yehezkiel1086/secure-go-api/db/migrations"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres"
)

func TestMigrations_EmbeddedContent(t *testing.T) {
	if migrations.Version == "" {
		t.Fatal("expected migration Version to be non-empty")
	}
	if len(migrations.SchemaSQL) == 0 {
		t.Fatal("expected SchemaSQL to be non-empty")
	}
	if !strings.Contains(migrations.SchemaSQL, "CREATE TABLE \"users\"") {
		t.Error("expected SchemaSQL to define users table")
	}
	if !strings.Contains(migrations.SchemaSQL, "CREATE TABLE \"refresh_tokens\"") {
		t.Error("expected SchemaSQL to define refresh_tokens table")
	}
	if !strings.Contains(migrations.SchemaSQL, "CREATE TABLE \"jobs\"") {
		t.Error("expected SchemaSQL to define jobs table")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	_ = godotenv.Load("../../../../.env")
	conf, err := config.New()
	if err != nil {
		t.Skip("skipping database test: .env not loaded")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, conf.DB)
	if err != nil {
		t.Skipf("skipping live database test: unable to connect: %v", err)
	}
	defer pool.Close()

	// Running migration should be completely idempotent
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}

	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatalf("second (idempotent) migration run failed: %v", err)
	}
}
