// Package repo is the Postgres persistence for tasks.
package repo

import (
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func marshalStringMapJSON(m map[string]string) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal string map: %w", err)
	}
	return b, nil
}

func marshalStringSliceJSON(s []string) ([]byte, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal string slice: %w", err)
	}
	return b, nil
}

type taskPgRepo struct {
	pool *pgxpool.Pool
}

func NewTaskPgRepo(pool *pgxpool.Pool) *taskPgRepo {
	return &taskPgRepo{pool: pool}
}
