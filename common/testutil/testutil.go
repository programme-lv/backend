package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/conf"
)

func MustGetMigratedTestPostgresDb(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pgtestConf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	rootPath := conf.FindProjectRoot()
	gm := golangmigrator.New(rootPath + "/migrate")
	config := pgtestdb.Custom(t, pgtestConf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}
