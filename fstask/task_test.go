package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var prjRootPath = filepath.Join(".", "..")
var testdataPath = filepath.Join(prjRootPath, "fstask", "testdata")
var tornisPath = filepath.Join(testdataPath, "tornis")
var kvadrputeklV2Dot5Path = filepath.Join(testdataPath, "kvadrputekl_v2dot5")

// writeAndReReadTask writes the given task to a temporary directory and reads it
// back from there. The temporary directory is removed after the function
// returns. The function is used in tests to check that a task can be written
// and read back correctly.
func writeAndReReadTask(t *testing.T, task *fstask.Task) *fstask.Task {
	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	t.Logf("Created directory for output: %s", outputDirectory)

	err = task.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	return storedTask
}

func TestReadingWritingTestGroups(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklV2Dot5Path)
	assert.NoErrorf(t, err, "failed to read task: %v", err)

	parsedTestGroups := parsedTask.GetTestGroupIDs()
	require.Equal(t, 2, len(parsedTestGroups))

	expectedTestGroups := []int{1, 2}
	assert.Equal(t, expectedTestGroups, parsedTestGroups)

	firstParsedTestGroup := parsedTask.GetInfoOnTestGroup(1)
	assert.Equal(t, 1, firstParsedTestGroup.GroupID)
	assert.Equal(t, 3, firstParsedTestGroup.Points)
	assert.Equal(t, 1, firstParsedTestGroup.Subtask)
	assert.Equal(t, true, firstParsedTestGroup.Public)
	assert.Equal(t, []int{1, 2, 3}, firstParsedTestGroup.TestIDs)

	assert.Equal(t, "kp01a", parsedTask.GetTestFilename(1))
	assert.Equal(t, "kp01b", parsedTask.GetTestFilename(2))
	assert.Equal(t, "kp01c", parsedTask.GetTestFilename(3))

	secondParsedTestGroup := parsedTask.GetInfoOnTestGroup(2)
	assert.Equal(t, 2, secondParsedTestGroup.GroupID)
	assert.Equal(t, 8, secondParsedTestGroup.Points)
	assert.Equal(t, 2, secondParsedTestGroup.Subtask)
	assert.Equal(t, false, secondParsedTestGroup.Public)
	assert.Equal(t, []int{4, 5, 6}, secondParsedTestGroup.TestIDs)

	assert.Equal(t, "kp02a", parsedTask.GetTestFilename(4))
	assert.Equal(t, "kp02b", parsedTask.GetTestFilename(5))
	assert.Equal(t, "kp02c", parsedTask.GetTestFilename(6))

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	writtenTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	writtenTestGroups := writtenTask.GetTestGroupIDs()
	require.Equal(t, 2, len(writtenTestGroups))

	firstWrittenTestGroup := writtenTask.GetInfoOnTestGroup(1)
	assert.Equal(t, 1, firstWrittenTestGroup.GroupID)
	assert.Equal(t, 3, firstWrittenTestGroup.Points)
	assert.Equal(t, 1, firstWrittenTestGroup.Subtask)
	assert.Equal(t, true, firstWrittenTestGroup.Public)
	assert.Equal(t, []int{1, 2, 3}, firstWrittenTestGroup.TestIDs)

	assert.Equal(t, "kp01a", writtenTask.GetTestFilename(1))
	assert.Equal(t, "kp01b", writtenTask.GetTestFilename(2))
	assert.Equal(t, "kp01c", writtenTask.GetTestFilename(3))

	secondWrittenTestGroup := writtenTask.GetInfoOnTestGroup(2)
	assert.Equal(t, 2, secondWrittenTestGroup.GroupID)
	assert.Equal(t, 8, secondWrittenTestGroup.Points)
	assert.Equal(t, 2, secondWrittenTestGroup.Subtask)
	assert.Equal(t, false, secondWrittenTestGroup.Public)
	assert.Equal(t, []int{4, 5, 6}, secondWrittenTestGroup.TestIDs)

	assert.Equal(t, "kp02a", writtenTask.GetTestFilename(4))
	assert.Equal(t, "kp02b", writtenTask.GetTestFilename(5))
	assert.Equal(t, "kp02c", writtenTask.GetTestFilename(6))

	createdTask, err := fstask.NewTask(writtenTask.GetTaskName())
	require.NoErrorf(t, err, "should have failed to create task: %v", err)

	createdTask.AddTestGroup(3, true, []int{7, 8, 9}, 1)

	assert.Equal(t, 1, createdTask.GetInfoOnTestGroup(1).GroupID)
	assert.Equal(t, 3, createdTask.GetInfoOnTestGroup(1).Points)
	assert.Equal(t, 1, createdTask.GetInfoOnTestGroup(1).Subtask)
	assert.Equal(t, true, createdTask.GetInfoOnTestGroup(1).Public)
	assert.Equal(t, []int{7, 8, 9}, createdTask.GetInfoOnTestGroup(1).TestIDs)
}
