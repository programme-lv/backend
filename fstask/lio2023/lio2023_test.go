package lio2023_test

import (
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask/lio2023"
	"github.com/stretchr/testify/require"
)

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

	examples := task.Examples
	require.Len(t, examples, 1)
	require.Equal(t, []byte("131\n"), examples[0].Input)
	require.Equal(t, []byte("1 131\n"), examples[0].Output)

	tests := task.Tests
	require.Len(t, tests, 4)

	require.Equal(t, []byte("560\n"), tests[2].Input)

	publicTestGroups := []int{1, 6, 11}
	testGroups := task.TestGroups
	require.Len(t, testGroups, 25)
	for i, testGroup := range testGroups {
		if testGroup.Public {
			require.Contains(t, publicTestGroups, i+1)
		}
	}

	require.Equal(t, 4, testGroups[0].Points)
	// require.Equal(t, 1, testGroups[1].Subtask) (can't be accurately determined)
	require.Equal(t, false, testGroups[1].Public)
	// require.Equal(t, true, testGroups[0].Public) (can't be accurately determined)
	require.Equal(t, []int{1, 2, 3, 4}, testGroups[0].TestIDs)

	require.Equal(t, 1.5, task.CPUTimeLimitSeconds)
	require.Equal(t, 256, task.MemoryLimitMegabytes)

	expectedArchive := []string{"./riki/interval.txt", "./riki/testlib.h"}
	actualArchive := []string{}
	for _, archiveFile := range task.ArchiveFiles {
		actualArchive = append(actualArchive, archiveFile.RelativePath)
	}

	require.ElementsMatch(t, expectedArchive, actualArchive)
}

func getTaskDirectory(t *testing.T, taskName string) (string, error) {
	testdataDirRel := filepath.Join("testdata", taskName)
	path, err := filepath.Abs(testdataDirRel)
	require.NoErrorf(t, err, "failed to get absolute path: %v", err)
	return path, nil
}
