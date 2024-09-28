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

	examples := task.GetExamples()
	require.Len(t, examples, 1)
	require.NotNilf(t, examples[0].FName, "examples[0].FName is nil")
	require.Equal(t, "00a", *examples[0].FName)
	require.Equal(t, []byte("131\n"), examples[0].Input)
	require.Equal(t, []byte("1 131\n"), examples[0].Output)

	require.Equal(t, "01a", task.GetTestFilename(1))
	require.Equal(t, "01b", task.GetTestFilename(2))
	require.Equal(t, "01c", task.GetTestFilename(3))
	require.Equal(t, "01d", task.GetTestFilename(4))

	tests := task.GetTestsSortedByID()
	require.Len(t, tests, 4)
	require.Equal(t, 1, tests[0].ID)
	require.Equal(t, 2, tests[1].ID)
	require.Equal(t, 3, tests[2].ID)
	require.Equal(t, 4, tests[3].ID)

	require.Equal(t, []byte("560\n"), tests[2].Input)

	publicTestGroups := []int{1, 6, 11}
	testGroups := task.GetTestGroups()
	require.Len(t, testGroups, 25)
	for _, testGroup := range testGroups {
		if testGroup.Public {
			require.Contains(t, publicTestGroups, testGroup.GroupID)
		}
	}

	require.Equal(t, 1, testGroups[0].GroupID)
	require.Equal(t, 4, testGroups[0].Points)
	require.Equal(t, 1, testGroups[1].Subtask)
	require.Equal(t, false, testGroups[1].Public)
	require.Equal(t, true, testGroups[0].Public)
	require.Equal(t, []int{1, 2, 3, 4}, testGroups[0].TestIDs)

	require.Equal(t, 1.5, task.CpuTimeLimInSeconds)
	require.Equal(t, 256, task.MemoryLimInMegabytes)

	expectedArchive := []string{"./riki/interval.txt", "./riki/testlib.h"}
	actualArchive := []string{}
	for _, archiveFile := range task.ArchiveFiles {
		actualArchive = append(actualArchive, archiveFile.RelativePath)
	}

	require.ElementsMatch(t, expectedArchive, actualArchive)
}

/*
1-1 4 1. apakšuzdevums --- PUBLISKA GRUPA
2-5 4
6-6 4 2. apakšuzdevums --- PUBLISKA GRUPA
7-10 4
11-11 4 3. apakšuzdevums --- PUBLISKA GRUPA
12-25 4
*/

func getTaskDirectory(t *testing.T, taskName string) (string, error) {
	testdataDirRel := filepath.Join("testdata", taskName)
	path, err := filepath.Abs(testdataDirRel)
	require.NoErrorf(t, err, "failed to get absolute path: %v", err)
	return path, nil
}
