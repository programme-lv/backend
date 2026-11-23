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

func TestAliasesReadWrite(t *testing.T) {
	ctx := context.Background()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	store := highlight.NewStore(pool)
	data, err := os.ReadFile("../repo/testdata/aplusbirc.json")
	require.NoError(t, err)
	var task srvc.Task
	require.NoError(t, json.Unmarshal(data, &task))
	require.NoError(t, repo.NewTaskPgRepo(pool).CreateTask(ctx, task))
	abbreviation, english := " DP ", " dynamic programming "
	a, err := store.Create(ctx, "dynamic-programming", "Dinamiskā programmēšana", highlight.Aliases{Abbreviation: &abbreviation, EnglishLabel: &english})
	require.NoError(t, err)
	require.Equal(t, "DP", a.Abbreviation)
	require.Equal(t, "dynamic programming", a.EnglishLabel)
	list, err := store.List(ctx)
	require.NoError(t, err)
	require.Equal(t, a, list[0])
	assignment, err := store.Replace(ctx, task.ShortId, []string{a.ID}, 0)
	require.NoError(t, err)
	want := highlight.Label{ID: a.ID, Label: a.Label, Abbreviation: "DP", EnglishLabel: "dynamic programming"}
	require.Equal(t, []highlight.Label{want}, assignment.HighlightedTags)
	read, err := store.Get(ctx, task.ShortId)
	require.NoError(t, err)
	require.Equal(t, assignment, read)
	batch, err := store.Batch(ctx, []string{task.ShortId})
	require.NoError(t, err)
	require.Equal(t, []highlight.Label{want}, batch[task.ShortId])
	empty := ""
	a, err = store.Edit(ctx, a.ID, a.Label, false, a.Revision, highlight.Aliases{Abbreviation: &empty})
	require.NoError(t, err)
	require.Empty(t, a.Abbreviation)
	require.Equal(t, "dynamic programming", a.EnglishLabel)
	newAbbreviation := "D"
	a, err = store.Edit(ctx, a.ID, a.Label, false, a.Revision, highlight.Aliases{Abbreviation: &newAbbreviation, EnglishLabel: &empty})
	require.NoError(t, err)
	require.Equal(t, "D", a.Abbreviation)
	require.Empty(t, a.EnglishLabel)
	a, err = store.Edit(ctx, a.ID, a.Label, false, a.Revision)
	require.NoError(t, err)
	require.Equal(t, "D", a.Abbreviation)
	invalid := "x\n"
	_, err = store.Edit(ctx, a.ID, a.Label, false, a.Revision, highlight.Aliases{EnglishLabel: &invalid})
	require.ErrorIs(t, err, highlight.ErrInvalid)
	_, err = store.Create(ctx, "invalid", "Invalid", highlight.Aliases{Abbreviation: &invalid})
	require.ErrorIs(t, err, highlight.ErrInvalid)
}

func TestAliasMigration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.MustGetMigratedTestPostgresDb(t)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)
	down, err := os.ReadFile("../../../postgres/migrate/081_task_highlight_aliases.down.sql")
	require.NoError(t, err)
	up, err := os.ReadFile("../../../postgres/migrate/081_task_highlight_aliases.up.sql")
	require.NoError(t, err)
	_, err = tx.Exec(ctx, string(down))
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO task_highlight_tags(id,label) VALUES ('dynamic-programming','DP label'), ('breadth-first-search','BFS label'), ('binary-search','binārmeklēšana')`)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, string(up))
	require.NoError(t, err)
	var label, abbreviation, english string
	var revision int
	require.NoError(t, tx.QueryRow(ctx, `SELECT label,abbreviation,english_label,revision FROM task_highlight_tags WHERE id='dynamic-programming'`).Scan(&label, &abbreviation, &english, &revision))
	require.Equal(t, "DP label", label)
	require.Equal(t, "DP", abbreviation)
	require.Equal(t, "dynamic programming", english)
	require.Equal(t, 2, revision)
	require.NoError(t, tx.QueryRow(ctx, `SELECT abbreviation,english_label FROM task_highlight_tags WHERE id='breadth-first-search'`).Scan(&abbreviation, &english))
	require.Equal(t, "BFS", abbreviation)
	require.Equal(t, "breadth-first search", english)
	require.NoError(t, tx.QueryRow(ctx, `SELECT label,revision FROM task_highlight_tags WHERE id='binary-search'`).Scan(&label, &revision))
	require.Equal(t, "binārā meklēšana", label)
	require.Equal(t, 2, revision)
	_, err = tx.Exec(ctx, string(down))
	require.NoError(t, err)
	require.NoError(t, tx.QueryRow(ctx, `SELECT label FROM task_highlight_tags WHERE id='binary-search'`).Scan(&label))
	require.Equal(t, "binārā meklēšana", label)
	_, err = tx.Exec(ctx, `DELETE FROM task_highlight_tags WHERE id <> 'binary-search'; UPDATE task_highlight_tags SET label='Custom', revision=9 WHERE id='binary-search'`)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, string(up))
	require.NoError(t, err)
	var count int
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM task_highlight_tags`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, tx.QueryRow(ctx, `SELECT label,revision FROM task_highlight_tags WHERE id='binary-search'`).Scan(&label, &revision))
	require.Equal(t, "Custom", label)
	require.Equal(t, 9, revision)
}
