package lio2023_test

import (
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/stretchr/testify/require"
)

func TestParsingLio2023TaskWithoutAChecker(t *testing.T) {
	// TODO: parse task "pbumbinas"
}

func TestParsingLio2023TaskWithAChecker(t *testing.T) {
	// TODO: parse task "zagas"
}

func TestParsingLio2023TaskWithAnInteractor(t *testing.T) {
	// TODO: parse task "pulkstenis"
}

func TestParsingLio2023TaskWithBothACheckerAndAnInteractor(t *testing.T) {
	taskDir, err := getTaskDirectory(t, "iedalas")
	require.NoErrorf(t, err, "failed to get task directory: %v", err)

	task, err := lio2023.ParseLio2023TaskDir(taskDir)
	require.NoErrorf(t, err, "failed to parse task: %v", err)

	require.NotNilf(t, task, "task is nil")

	require.NotNilf(t, task.TestlibChecker, "task.TestlibChecker is nil")
	require.NotNilf(t, task.TestlibInteractor, "task.TestlibInteractor is nil")

	require.Len(t, task.Solutions, 13)
	solutionFilenames := []string{}
	for _, solution := range task.Solutions {
		solutionFilenames = append(solutionFilenames, solution.Filename)
	}
	require.Contains(t, solutionFilenames, "iedalas_PP_OK.cpp")
}

func getTaskDirectory(t *testing.T, taskName string) (string, error) {
	testdataDirRel := filepath.Join("testdata", taskName)
	path, err := filepath.Abs(testdataDirRel)
	require.NoErrorf(t, err, "failed to get absolute path: %v", err)
	return path, nil
}
