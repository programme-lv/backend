package repo_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
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
	gm := golangmigrator.New("../../postgres/migrate")
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
	// parse it into a tasksrvc.Task struct
	// insert the task into the database
	// read the task from the database
	// compare the task with manual inspection

	pool := NewDB(t)
	repo := repo.NewTaskPgRepo(pool)

	taskJson, err := os.ReadFile("testdata/aplusbirc.json")
	if err != nil {
		t.Fatalf("Failed to read testdata/aplusbirc.json: %v", err)
	}

	var task srvc.Task
	err = json.Unmarshal(taskJson, &task)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	ctx := context.Background()

	// Test task creation
	err = repo.CreateTask(ctx, task)
	require.NoError(t, err, "Failed to create task")
	_, err = pool.Exec(ctx, `
		UPDATE task_origin_notes
		SET info_short = $2
		WHERE task_short_id = $1 AND lang = 'lv'
	`, task.ShortId, "LIO 38. atlases kārta")
	require.NoError(t, err, "Failed to update task origin note short text")
	// Test task existence
	exists, err := repo.Exists(ctx, task.ShortId)
	require.NoError(t, err, "Failed to check if task exists")
	assert.True(t, exists, "Task should exist after creation")

	// Test retrieving the task
	retrievedTask, err := repo.GetTask(ctx, task.ShortId)
	require.NoError(t, err, "Failed to get task")

	// Verify task fields
	assert.Equal(t, "aplusbirc", retrievedTask.ShortId, "ShortId mismatch")
	assert.Equal(t, "A+B=C", retrievedTask.DefaultFullName(), "FullName mismatch")
	assert.Equal(t, "task-md-images/nekoks.png", retrievedTask.IllustrImg.S3Key, "IllustrImgS3Key mismatch")
	assert.Equal(t, 256, retrievedTask.MemLimMegabytes, "MemLimMegabytes mismatch")
	assert.Equal(t, 0.6, retrievedTask.CpuTimeLimSecs, "CpuTimeLimSecs mismatch")
	assert.Equal(t, "LIO", retrievedTask.OriginOlympiad, "OriginOlympiad mismatch")
	assert.Equal(t, "PPS", retrievedTask.OriginOrg, "OriginOrg mismatch")
	assert.Equal(t, "2024/2025", retrievedTask.OriginYear, "OriginYear mismatch")
	assert.Contains(t, retrievedTask.ProblemTags, "two-sum", "ProblemTags missing two-sum")
	assert.Equal(t, []string{"Kaspars", "Pēteris"}, retrievedTask.Authors, "Authors mismatch")
	assert.Equal(t, "task-archives/aplusbirc.zip", retrievedTask.OgFilesZipS3Key, "ArchiveS3Key mismatch")
	assert.Equal(t, 3, retrievedTask.DifficultyRating, "DifficultyRating mismatch")
	assert.Contains(t, retrievedTask.Checker, "#include", "Checker mismatch")
	assert.Equal(t, "", retrievedTask.Interactor, "Interactor mismatch")
	assert.Equal(t, "some markdown content", retrievedTask.Readme, "Readme mismatch")

	// Verify nested structures
	assert.Len(t, retrievedTask.OriginNotes, 1, "OriginNotes length mismatch")
	assert.Contains(t, retrievedTask.OriginNotes[0].Info, "Uzdevums no Latvijas 38.", "OriginNotes content mismatch")
	assert.Equal(t, "lv", retrievedTask.OriginNotes[0].Lang, "OriginNotes language mismatch")

	assert.Len(t, retrievedTask.MdStatements, 1, "MdStatements length mismatch")
	assert.Equal(t, "lv", retrievedTask.MdStatements[0].LangIso639, "MdStatements language mismatch")
	assert.Contains(t, retrievedTask.MdStatements[0].Story, "Dotas $N$ kartītes", "MdStatements story mismatch")

	assert.Len(t, retrievedTask.PdfStatements, 1, "PdfStatements length mismatch")
	assert.Contains(t, retrievedTask.PdfStatements[0].S3Key, "task-pdf-statements/7a25c752637f3b913bac77e962e80c153b52caf1cd824f4b81da0c31df7f5f19.pdf", "PdfStatements S3Key mismatch")

	assert.Len(t, retrievedTask.VisInpSubtasks, 1, "VisInpSubtasks length mismatch")
	assert.Len(t, retrievedTask.VisInpSubtasks[0].Tests, 3, "VisInpSubtasks tests length mismatch")

	assert.Len(t, retrievedTask.Examples, 2, "Examples length mismatch")
	assert.Contains(t, retrievedTask.Examples[0].Input, "1 3 6 3 -1 4", "Example input mismatch")
	assert.Contains(t, retrievedTask.Examples[1].Output, "0", "Example output mismatch")
	assert.Contains(t, retrievedTask.Examples[1].MdNote["en"], "Uzdevuma tekstā dotie trīs testi")

	assert.Len(t, retrievedTask.Tests, 1, "Tests length mismatch")
	assert.Contains(t, retrievedTask.Tests[0].InpSha2, "c21d04a1cb0bc201602720f10cbdda6319140e031de2b9753509f589a63d4339", "Test hash mismatch")

	assert.Len(t, retrievedTask.Subtasks, 5, "Subtasks length mismatch")
	assert.Equal(t, 2, retrievedTask.Subtasks[0].Score, "First subtask score mismatch")
	assert.Contains(t, retrievedTask.Subtasks[0].Descriptions["lv"], "Uzdevuma tekstā dotie trīs testi", "Subtask description mismatch")

	assert.Len(t, retrievedTask.TestGroups, 21, "TestGroups length mismatch")
	assert.Equal(t, 2, retrievedTask.TestGroups[0].Points, "First test group points mismatch")
	assert.True(t, retrievedTask.TestGroups[0].Public, "First test group should be public")

	// Verify Solutions
	assert.Len(t, retrievedTask.Solutions, 2, "Solutions length mismatch")

	solutionMap := make(map[string]srvc.Solution)
	for _, sol := range retrievedTask.Solutions {
		solutionMap[sol.Fname] = sol
	}

	cppSol := solutionMap["solution.cpp"]
	assert.Equal(t, "solution.cpp", cppSol.Fname, "C++ solution filename")
	assert.Contains(t, cppSol.Content, "#include <iostream>", "C++ solution content")
	assert.Equal(t, []int{1, 2, 3, 4, 5}, cppSol.Subtasks, "C++ solution subtasks")

	pySol := solutionMap["partial.py"]
	assert.Equal(t, "partial.py", pySol.Fname, "Python solution filename")
	assert.Equal(t, []int{1, 2}, pySol.Subtasks, "Python solution subtasks")

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

	// Test GetTaskPreview
	t.Run("GetTaskPreview", func(t *testing.T) {
		// Test getting task preview
		taskPreview, err := repo.GetTaskPreview(ctx, task.ShortId)
		require.NoError(t, err, "Failed to get task preview")

		// Verify basic task preview fields
		assert.Equal(t, "aplusbirc", taskPreview.ShortId, "Preview ShortId mismatch")
		assert.Equal(t, "A+B=C", taskPreview.DefaultFullName(), "Preview FullName mismatch")
		assert.Equal(t, "task-md-images/nekoks.png", taskPreview.IllustrImg.S3Key, "Preview IllustrImgS3Key mismatch")
		assert.Equal(t, "LIO", taskPreview.OriginOlympiad, "Preview OriginOlympiad mismatch")
		assert.Equal(t, "", taskPreview.OriginOrg, "Preview OriginOrg mismatch")
		assert.Equal(t, "", taskPreview.OriginYear, "Preview OriginYear mismatch")
		assert.Equal(t, 3, taskPreview.DifficultyRating, "Preview DifficultyRating mismatch")

		// Verify illustration image fields
		assert.NotNil(t, taskPreview.IllustrImg.WidthPx, "Preview WidthPx should not be nil")
		assert.NotNil(t, taskPreview.IllustrImg.HeightPx, "Preview HeightPx should not be nil")
		assert.NotNil(t, taskPreview.IllustrImg.SzInBytes, "Preview SzInBytes should not be nil")

		// Verify origin notes
		assert.Contains(t, taskPreview.OriginNote, "Uzdevums no Latvijas 38.", "Preview OriginNote mismatch")
		assert.Equal(t, "LIO 38. atlases kārta", taskPreview.OriginNoteShort, "Preview OriginNoteShort mismatch")

		// Verify markdown statement story
		assert.Contains(t, taskPreview.MdStatementStory, "Dotas $N$ kartītes", "Preview MdStatementStory mismatch")

		// Test getting preview for non-existent task
		_, err = repo.GetTaskPreview(ctx, "non-existent-task")
		assert.Error(t, err, "Getting preview for non-existent task should fail")
		assert.Contains(t, err.Error(), "failed to load task preview", "Error message should indicate task preview loading failed")
	})

	t.Run("ListTaskPreviews", func(t *testing.T) {
		taskPreviews, err := repo.ListTaskPreviews(ctx, 10, 0)
		require.NoError(t, err, "Failed to list task previews")
		require.Len(t, taskPreviews, 1, "Should have exactly one task preview")
		assert.Equal(t, "LIO 38. atlases kārta", taskPreviews[0].OriginNoteShort, "Listed preview OriginNoteShort mismatch")
	})

	// Test DeleteTask
	t.Run("DeleteTask", func(t *testing.T) {
		// First verify the task exists
		exists, err := repo.Exists(ctx, task.ShortId)
		require.NoError(t, err, "Failed to check if task exists before deletion")
		assert.True(t, exists, "Task should exist before deletion")

		// Verify solutions exist before deletion
		taskBeforeDeletion, err := repo.GetTask(ctx, task.ShortId)
		require.NoError(t, err, "Failed to get task before deletion")
		assert.Len(t, taskBeforeDeletion.Solutions, 2, "Should have solutions before deletion")

		// Delete the task
		err = repo.DeleteTask(ctx, task.ShortId)
		require.NoError(t, err, "Failed to delete task")

		// Verify the task no longer exists
		exists, err = repo.Exists(ctx, task.ShortId)
		require.NoError(t, err, "Failed to check if task exists after deletion")
		assert.False(t, exists, "Task should not exist after deletion")

		// Verify solutions are also deleted (check directly in database)
		var solutionCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM task_solutions WHERE task_id = $1", task.ShortId).Scan(&solutionCount)
		require.NoError(t, err, "Failed to count solutions after deletion")
		assert.Equal(t, 0, solutionCount, "All solutions should be deleted")

		// Try to get the deleted task (should fail)
		_, err = repo.GetTask(ctx, task.ShortId)
		assert.Error(t, err, "Getting deleted task should fail")

		// Try to delete a non-existent task (should fail)
		err = repo.DeleteTask(ctx, "non-existent-task")
		assert.Error(t, err, "Deleting non-existent task should fail")
		assert.Contains(t, err.Error(), "does not exist", "Error message should indicate task does not exist")

		// Verify we can't list the deleted task
		tasks, err := repo.ListTasks(ctx, 10, 0)
		require.NoError(t, err, "Failed to list tasks after deletion")
		assert.Len(t, tasks, 0, "Should have no tasks after deletion")
	})

}
