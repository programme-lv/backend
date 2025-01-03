package submsrvc

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/stretchr/testify/assert"
)

// NewDB returns an open connection to a unique and isolated test database,
// fully migrated and ready for testing
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	gm := golangmigrator.New("../migrate")
	return pgtestdb.New(t, conf, gm)
}

func TestPgDbSchemaVersion(t *testing.T) {
	t.Parallel()

	db := NewDB(t)

	var version int
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	assert.Nil(t, err)
	assert.Equal(t, 22, version)
	assert.False(t, dirty)
}
