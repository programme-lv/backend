package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/conf"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// MustGetMigratedTestPostgresDb returns an isolated, migrated Postgres pool.
// Defaults match postgres/compose.yml; override with TEST_PG_*.
func MustGetMigratedTestPostgresDb(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pgtestConf := pgtestdb.Config{
		DriverName: "pgx",
		User:       envOr("TEST_PG_USER", "postgres"),
		Password:   envOr("TEST_PG_PASSWORD", "pw"),
		Host:       envOr("TEST_PG_HOST", "localhost"),
		Port:       envOr("TEST_PG_PORT", "5432"),
		Options:    "sslmode=disable",
	}
	migrateDir := filepath.Join(conf.FindProjectRoot(), "postgres", "migrate")
	gm := golangmigrator.New(migrateDir)
	config := pgtestdb.Custom(t, pgtestConf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	if err != nil {
		t.Fatalf("create connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}
