//go:build integration

package highlight_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/modules/task/highlight"
	"github.com/programme-lv/backend/modules/task/repo"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/stretchr/testify/require"
)

func TestPersistenceAndConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	store := highlight.NewStore(pool)
	data, err := os.ReadFile("../repo/testdata/aplusbirc.json")
	require.NoError(t, err)
	var task srvc.Task
	require.NoError(t, json.Unmarshal(data, &task))
	tasks := repo.NewTaskPgRepo(pool)
	require.NoError(t, tasks.CreateTask(ctx, task))
	var originalRevision, originalTestsVersion int
	require.NoError(t, pool.QueryRow(ctx, `SELECT revision, tests_version FROM tasks WHERE short_id=$1`, task.ShortId).Scan(&originalRevision, &originalTestsVersion))
	initial, err := store.Get(ctx, task.ShortId)
	require.NoError(t, err)
	require.Equal(t, int64(0), initial.Revision)
	require.Empty(t, initial.HighlightedTags)
	a, err := store.Create(ctx, "alpha", "  Ābols ")
	require.NoError(t, err)
	require.Equal(t, highlight.Tag{ID: "alpha", Label: "Ābols", Revision: 1}, a)
	_, err = store.Create(ctx, "alpha", "x")
	require.ErrorIs(t, err, highlight.ErrConflict)
	_, err = store.Create(ctx, "beta", "Beta")
	require.NoError(t, err)
	for _, ids := range [][]string{nil, {"alpha", "alpha"}, {"alpha", "beta", "gamma"}, {"unknown"}} {
		_, err = store.Replace(ctx, task.ShortId, ids, 0)
		require.ErrorIs(t, err, highlight.ErrInvalid)
	}
	assigned, err := store.Replace(ctx, task.ShortId, []string{"beta", "alpha"}, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), assigned.Revision)
	require.Equal(t, "beta", assigned.HighlightedTags[0].ID)
	_, err = store.Replace(ctx, task.ShortId, []string{}, 0)
	require.ErrorIs(t, err, highlight.ErrConflict)
	a, err = store.Edit(ctx, "alpha", "New", true, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), a.Revision)
	_, err = store.Edit(ctx, "alpha", "Other", false, 1)
	require.ErrorIs(t, err, highlight.ErrConflict)
	_, err = store.Edit(ctx, "alpha", "Other", false, 0)
	require.ErrorIs(t, err, highlight.ErrConflict)
	assigned, err = store.Get(ctx, task.ShortId)
	require.NoError(t, err)
	require.Equal(t, "New", assigned.HighlightedTags[1].Label)
	assigned, err = store.Replace(ctx, task.ShortId, []string{"alpha"}, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), assigned.Revision)
	_, err = store.Replace(ctx, task.ShortId, []string{}, 2)
	require.NoError(t, err)
	_, err = store.Replace(ctx, task.ShortId, []string{"alpha"}, 3)
	require.ErrorIs(t, err, highlight.ErrInvalid)
	_, err = store.Replace(ctx, task.ShortId, []string{"beta"}, 3)
	require.NoError(t, err)
	var taskRev, testsVersion int
	require.NoError(t, pool.QueryRow(ctx, `SELECT revision, tests_version FROM tasks WHERE short_id=$1`, task.ShortId).Scan(&taskRev, &testsVersion))
	require.Equal(t, originalRevision, taskRev)
	require.Equal(t, originalTestsVersion, testsVersion)
	require.NoError(t, tasks.UpdateTask(ctx, task, false, taskRev))
	var revAfter, testsAfter int
	require.NoError(t, pool.QueryRow(ctx, `SELECT revision, tests_version FROM tasks WHERE short_id=$1`, task.ShortId).Scan(&revAfter, &testsAfter))
	require.Equal(t, testsVersion, testsAfter)
	current, err := store.Get(ctx, task.ShortId)
	require.NoError(t, err)
	require.Equal(t, int64(4), current.Revision)
	require.Equal(t, []highlight.Label{{ID: "beta", Label: "Beta"}}, current.HighlightedTags)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := store.Replace(ctx, task.ShortId, []string{"beta"}, 4); results <- e }()
	}
	wg.Wait()
	close(results)
	ok, conflicts := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if e == highlight.ErrConflict {
			conflicts++
		} else {
			t.Error(e)
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, conflicts)
	task.ShortId = "second-task"
	require.NoError(t, tasks.CreateTask(ctx, task))
	results = make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := store.Replace(ctx, task.ShortId, []string{"beta"}, 0)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	ok, conflicts = 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if e == highlight.ErrConflict {
			conflicts++
		} else {
			t.Error(e)
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, conflicts)
	labels, err := store.Batch(ctx, []string{task.ShortId})
	require.NoError(t, err)
	require.Equal(t, []highlight.Label{{ID: "beta", Label: "Beta"}}, labels[task.ShortId])
	list, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.True(t, list[0].Archived)
}
