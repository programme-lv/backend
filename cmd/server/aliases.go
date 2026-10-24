package main

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/jsonresp"
)

// taskAliasResolver is loaded once because renames require an offline
// maintenance run and a backend restart. It keeps image requests DB-free.
type taskAliasResolver struct {
	mu      sync.RWMutex
	aliases map[string]string
}

func newTaskAliasResolver(ctx context.Context, pool *pgxpool.Pool) (*taskAliasResolver, error) {
	rows, err := pool.Query(ctx, `SELECT alias, task_id FROM task_aliases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := &taskAliasResolver{aliases: make(map[string]string)}
	for rows.Next() {
		var alias, taskID string
		if err := rows.Scan(&alias, &taskID); err != nil {
			return nil, err
		}
		r.aliases[alias] = taskID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *taskAliasResolver) Resolve(_ context.Context, id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if canonical, ok := r.aliases[id]; ok {
		return canonical, nil
	}
	return id, nil
}

func taskAliases(resolve func(context.Context, string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(r.URL.Path, "/")
			index := -1
			if len(parts) > 2 && parts[1] == "tasks" {
				index = 2
			}
			if len(parts) > 3 && parts[1] == "assets" && parts[2] == "tasks" {
				index = 3
			}
			if index >= 0 {
				id, err := resolve(r.Context(), parts[index])
				if err != nil {
					jsonresp.InternalError(w)
					return
				}
				if id != parts[index] {
					parts[index] = id
					u := *r.URL
					u.Path = strings.Join(parts, "/")
					u.RawPath = ""
					http.Redirect(w, r, u.RequestURI(), http.StatusPermanentRedirect)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
