package taskpgrepo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/task/taskdomain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewDB returns a connection pool to a unique and isolated test database,
// fully migrated and ready for testing
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	gm := golangmigrator.New("../../migrate")
	config := pgtestdb.Custom(t, conf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestTaskPgRepo(t *testing.T) {
	// read the testdata/aplusbirc.json file
	// parse it into a taskdomain.Task struct
	// insert the task into the database
	// read the task from the database
	// compare the task with manual inspection

	pool := NewDB(t)
	repo := NewTaskPgRepo(pool)

	taskJson, err := os.ReadFile("testdata/aplusbirc.json")
	if err != nil {
		t.Fatalf("Failed to read testdata/aplusbirc.json: %v", err)
	}

	var task taskdomain.Task
	err = json.Unmarshal(taskJson, &task)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	ctx := context.Background()

	// Test task creation
	err = repo.CreateTask(ctx, task)
	require.NoError(t, err, "Failed to create task")
	// Test task existence
	exists, err := repo.Exists(ctx, task.ShortId)
	require.NoError(t, err, "Failed to check if task exists")
	assert.True(t, exists, "Task should exist after creation")

	// Test retrieving the task
	retrievedTask, err := repo.GetTask(ctx, task.ShortId)
	require.NoError(t, err, "Failed to get task")

	// Verify task fields
	assert.Equal(t, "aplusbirc", retrievedTask.ShortId, "ShortId mismatch")
	assert.Equal(t, "A+B=C", retrievedTask.FullName, "FullName mismatch")
	assert.Equal(t, "", retrievedTask.IllustrImgUrl, "IllustrImgUrl mismatch")
	assert.Equal(t, 256, retrievedTask.MemLimMegabytes, "MemLimMegabytes mismatch")
	assert.Equal(t, 0.6, retrievedTask.CpuTimeLimSecs, "CpuTimeLimSecs mismatch")
	assert.Equal(t, "LIO", retrievedTask.OriginOlympiad, "OriginOlympiad mismatch")
	assert.Equal(t, 3, retrievedTask.DifficultyRating, "DifficultyRating mismatch")
	assert.Contains(t, retrievedTask.Checker, "#include", "Checker mismatch")
	assert.Equal(t, "", retrievedTask.Interactor, "Interactor mismatch")

	// Verify nested structures
	assert.Len(t, retrievedTask.OriginNotes, 1, "OriginNotes length mismatch")
	assert.Contains(t, retrievedTask.OriginNotes[0].Info, "Uzdevums no Latvijas 38.", "OriginNotes content mismatch")
	assert.Equal(t, "lv", retrievedTask.OriginNotes[0].Lang, "OriginNotes language mismatch")

	assert.Len(t, retrievedTask.MdStatements, 1, "MdStatements length mismatch")
	assert.Equal(t, "lv", retrievedTask.MdStatements[0].LangIso639, "MdStatements language mismatch")
	assert.Contains(t, retrievedTask.MdStatements[0].Story, "Dotas $N$ kartītes", "MdStatements story mismatch")

	assert.Len(t, retrievedTask.PdfStatements, 1, "PdfStatements length mismatch")
	assert.Contains(t, retrievedTask.PdfStatements[0].ObjectUrl, "proglv-public.s3.eu-central-1.amazonaws.com", "PdfStatements URL mismatch")

	assert.Len(t, retrievedTask.VisInpSubtasks, 1, "VisInpSubtasks length mismatch")
	assert.Len(t, retrievedTask.VisInpSubtasks[0].Tests, 3, "VisInpSubtasks tests length mismatch")

	assert.Len(t, retrievedTask.Examples, 2, "Examples length mismatch")
	assert.Contains(t, retrievedTask.Examples[0].Input, "1 3 6 3 -1 4", "Example input mismatch")
	assert.Contains(t, retrievedTask.Examples[1].Output, "0", "Example output mismatch")

	assert.Len(t, retrievedTask.Tests, 1, "Tests length mismatch")
	assert.Contains(t, retrievedTask.Tests[0].InpSha2, "c21d04a1cb0bc201602720f10cbdda6319140e031de2b9753509f589a63d4339", "Test hash mismatch")

	assert.Len(t, retrievedTask.Subtasks, 5, "Subtasks length mismatch")
	assert.Equal(t, 2, retrievedTask.Subtasks[0].Score, "First subtask score mismatch")
	assert.Contains(t, retrievedTask.Subtasks[0].Descriptions["lv"], "Uzdevuma tekstā dotie trīs testi", "Subtask description mismatch")

	assert.Len(t, retrievedTask.TestGroups, 21, "TestGroups length mismatch")
	assert.Equal(t, 2, retrievedTask.TestGroups[0].Points, "First test group points mismatch")
	assert.True(t, retrievedTask.TestGroups[0].Public, "First test group should be public")

	// Test ResolveNames
	names, err := repo.ResolveNames(ctx, []string{task.ShortId})
	require.NoError(t, err, "Failed to resolve names")
	assert.Equal(t, []string{"A+B=C"}, names, "ResolveNames returned incorrect result")

	// Test ListTasks
	tasks, err := repo.ListTasks(ctx, 10, 0)
	require.NoError(t, err, "Failed to list tasks")
	assert.Len(t, tasks, 1, "Should have exactly one task")
	assert.Equal(t, "aplusbirc", tasks[0].ShortId, "Listed task has incorrect ShortId")

	// Test creating a duplicate task (should fail)
	err = repo.CreateTask(ctx, task)
	assert.Error(t, err, "Creating duplicate task should fail")
	assert.Contains(t, err.Error(), "already exists", "Error message should indicate task already exists")

}
