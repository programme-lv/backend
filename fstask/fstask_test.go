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
var kvadrputeklPath = filepath.Join(testdataPath, "kvadrputekl")

func TestKvadrputeklTask(t *testing.T) {
	task, err := fstask.Read(kvadrputeklPath)
	require.NoError(t, err)

	requireKvadrputekl(t, task)

	task2 := writeAndReReadTask(t, task)

	requireKvadrputekl(t, task2)
}

func requireKvadrputekl(t *testing.T, task *fstask.Task) {
	// ARCHIVE
	require.Len(t, task.ArchiveFiles, 4)
	require.Contains(t, task.ArchiveFiles, fstask.ArchiveFile{
		RelativePath: "riki/hello.txt",
		Content:      []byte("hello"),
	})

	// ASSETS
	require.Len(t, task.Assets, 4)
	require.Contains(t, task.Assets, fstask.AssetFile{
		RelativePath: "test.txt",
		Content:      []byte("test"),
	})

	// EXAMPLES
	require.Len(t, task.Examples, 1)
	require.Contains(t, task.Examples, fstask.Example{
		Input:  []byte(`5 9 3`),
		Output: []byte(`10`),
		MdNote: []byte(`asdf`),
	})

	// SOLUTIONS
	require.Len(t, task.Solutions, 3)
	require.Equal(t, fstask.Solution{
		Filename: "kp_kp_ok.cpp",
		ScoreEq:  intPtr(100),
		ScoreLt:  nil,
		ScoreLte: nil,
		ScoreGt:  nil,
		ScoreGte: nil,
		Author:   strPtr("Krišjānis Petručeņa"),
		ExecTime: float64Ptr(0.035),
		Content:  []byte("#include <iostream>"),
	}, task.Solutions[0])

	// STATEMENTS
	require.Contains(t, task.MarkdownStatements, fstask.MarkdownStatement{
		Language: "lv",
		Story:    "![1. attēls: Laukuma piemērs](kp1.png)",
		Input:    "Ievaddati",
		Output:   "Izvaddati",
		Notes:    "",
		Scoring:  "",
	})
	require.Contains(t, task.MarkdownStatements, fstask.MarkdownStatement{
		Language: "en",
		Story:    "story",
		Input:    "input",
		Output:   "output",
		Notes:    "",
		Scoring:  "",
	})
	require.Len(t, task.PdfStatements, 1)
	require.Equal(t, "lv", task.PdfStatements[0].Language)
	require.NotEmpty(t, task.PdfStatements[0])

	// TESTS
	require.Len(t, task.Tests, 6)

	// SUBTASKS
	require.Len(t, task.Subtasks, 2)
	require.Equal(t, fstask.Subtask{
		Points:  48,
		TestIDs: []int{4, 5, 6},
		Descriptions: map[string]string{
			"lv": "$$NM \\leq 10^3$$",
			"en": "$$NM \\leq 10^3$$",
		},
	}, task.Subtasks[1])

	// TEST GROUPS
	require.Len(t, task.TestGroups, 2)
	require.Equal(t, fstask.TestGroup{
		Points:  3,
		Public:  true,
		TestIDs: []int{1, 2, 3},
	}, task.TestGroups[0])
	require.Equal(t, fstask.TestGroup{
		Points:  8,
		Public:  false,
		TestIDs: []int{4, 5, 6},
	}, task.TestGroups[1])

	// GENERAL
	require.Equal(t, "Kvadrātveida putekļsūcējs", task.FullName)
	require.Equal(t, []int{1}, task.VisibleInputSubtasks)
	require.Equal(t, "illustration.png", task.IllustrAssetFilename)
	require.Equal(t, []string{"bfs", "grid", "prefix-sum", "sliding-window", "shortest-path", "graphs"}, task.ProblemTags)
	require.Equal(t, 3, task.DifficultyOneToFive)

	// CONSTRAINTS
	require.Equal(t, 0.5, task.CPUTimeLimitSeconds)
	require.Equal(t, 256, task.MemoryLimitMegabytes)

	// ORIGIN
	require.Equal(t, "LIO", task.OriginOlympiad)
	require.Equal(t, "2023/2024", task.AcademicYear)
	require.Equal(t, "school", task.OlympiadStage)
	require.Equal(t, "", task.OriginInstitution)
	require.Equal(t, []string{"Krišjānis Petručeņa"}, task.TaskAuthors)
	require.Equal(t, map[string]string{
		"lv": "Uzdevums no Latvijas 37. informātikas olimpiādes (2023./2024. mācību gads) skolas kārtas.",
		"en": "The problem is from the school round of the 37th Latvian Informatics Olympiad in the 2023/2024 academic year.",
	}, task.OriginNotes)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

// writeAndReReadTask writes the given task to a temporary directory and reads it
// back from there. The temporary directory is removed after the function
// returns. The function is used in tests to check that a task can be written
// and read back correctly.
func writeAndReReadTask(t *testing.T, task *fstask.Task) *fstask.Task {
	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "task")

	err = task.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	return storedTask
}
