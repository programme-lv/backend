package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

var prjRootPath = filepath.Join(".", "..")
var testdataPath = filepath.Join(prjRootPath, "fstask", "testdata")
var kvadrputeklV3Dot0Path = filepath.Join(testdataPath, "kvadrputekl_v3dot0")

// writeAndReReadTask writes the given task to a temporary directory and reads it
// back from there. The temporary directory is removed after the function
// returns. The function is used in tests to check that a task can be written
// and read back correctly.
func writeAndReReadTask(t *testing.T, task *fstask.Task) *fstask.Task {
	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")

	err = task.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	return storedTask
}
