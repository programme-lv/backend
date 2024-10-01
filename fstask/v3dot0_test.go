package fstask_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestTestV3Dot0Spec(t *testing.T) {
	task, err := readTestdataTask("kvadrputekl_v3dot0")
	require.NoErrorf(t, err, "failed to read task: %v", err)

	require.Len(t, task.ArchiveFiles, 7)
	require.Len(t, task.Assets, 1)
	require.Len(t, task.GetExamples(), 1)
	require.Len(t, task.Solutions, 3)
	require.Len(t, task.GetTestsSortedByID(), 2)
	statements := task.MarkdownStatements
	require.Len(t, statements, 1)
	require.NotEmpty(t, statements[0].Story)
}

func readTestdataTask(taskDirName string) (*fstask.Task, error) {
	path := filepath.Join(testdataPath, taskDirName)
	task, err := fstask.Read(path)
	if err != nil {
		return nil, fmt.Errorf("error reading task: %w", err)
	}
	return task, nil
}
