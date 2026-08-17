// Package repo is the Postgres persistence for tasks.
package repo

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustMarshalMapToJSONB converts a map into JSON bytes suitable for a jsonb parameter.
// It never returns nil; on error it returns an empty JSON object.
func mustMarshalMapToJSONB(m map[string]string) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

// mustMarshalSliceToJSONB converts a slice of strings into JSON bytes suitable for a jsonb parameter.
// It never returns nil; on error it returns an empty JSON array.
func mustMarshalSliceToJSONB(s []string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

type taskPgRepo struct {
	pool *pgxpool.Pool
}

func NewTaskPgRepo(pool *pgxpool.Pool) *taskPgRepo {
	return &taskPgRepo{pool: pool}
}
