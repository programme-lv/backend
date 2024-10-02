package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingMetadata(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklV2Dot5Path)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	parsedTask.SetTaskName("Kvadrātveida putekļsūcējs")
	parsedTask.ProblemTags = []string{"math", "geometry"}
	parsedTask.TaskAuthors = []string{"Author1", "Author2"}
	parsedTask.OriginOlympiad = "LIO"
	parsedTask.DifficultyOneToFive = 3

	assert.Equal(t, "Kvadrātveida putekļsūcējs", parsedTask.GetTaskName())
	assert.Equal(t, []string{"math", "geometry"}, parsedTask.ProblemTags)
	assert.Equal(t, []string{"Author1", "Author2"}, parsedTask.TaskAuthors)
	assert.Equal(t, "LIO", parsedTask.OriginOlympiad)
	assert.Equal(t, 3, parsedTask.DifficultyOneToFive)
	require.Equal(t, map[string]string{
		"lv": "Uzdevums parādījās Latvijas 37. informātikas olimpiādes (2023./2024. gads) skolas kārtā.",
	}, parsedTask.GetOriginNotes())

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	assert.Equal(t, "Kvadrātveida putekļsūcējs", storedTask.GetTaskName())
	assert.Equal(t, []string{"math", "geometry"}, storedTask.ProblemTags)
	assert.Equal(t, []string{"Author1", "Author2"}, storedTask.TaskAuthors)
	assert.Equal(t, "LIO", storedTask.OriginOlympiad)
	assert.Equal(t, 3, storedTask.DifficultyOneToFive)
	require.Equal(t, map[string]string{
		"lv": "Uzdevums parādījās Latvijas 37. informātikas olimpiādes (2023./2024. gads) skolas kārtā.",
	}, storedTask.GetOriginNotes())
}
