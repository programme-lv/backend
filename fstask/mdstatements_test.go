package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingMDStatements(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklV2Dot5Path)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// compare markdown story to parsed one
	mdLvDir := filepath.Join(kvadrputeklV2Dot5Path, "statements", "md", "lv")
	InputMdPath := filepath.Join(mdLvDir, "input.md")
	OutputMdPath := filepath.Join(mdLvDir, "output.md")
	StoryMdPath := filepath.Join(mdLvDir, "story.md")
	ScoringMDPath := filepath.Join(mdLvDir, "scoring.md")

	inputMdBytes, err := os.ReadFile(InputMdPath)
	require.NoErrorf(t, err, "failed to read input.md file: %v", err)
	inputMd := string(inputMdBytes)

	outputMdBytes, err := os.ReadFile(OutputMdPath)
	require.NoErrorf(t, err, "failed to read output.md file: %v", err)
	outputMd := string(outputMdBytes)

	storyMdBytes, err := os.ReadFile(StoryMdPath)
	require.NoErrorf(t, err, "failed to read story.md file: %v", err)
	storyMd := string(storyMdBytes)

	scoringMdBytes, err := os.ReadFile(ScoringMDPath)
	require.NoErrorf(t, err, "failed to read scoring.md file: %v", err)
	scoringMd := string(scoringMdBytes)

	parsedMdStatements := parsedTask.MarkdownStatements
	require.Equal(t, 1, len(parsedMdStatements))
	for _, mdStatement := range parsedMdStatements {
		lang := mdStatement.Language
		require.NotNil(t, lang)
		require.Equal(t, "lv", lang)

		require.Equal(t, inputMd, mdStatement.Input)
		require.Equal(t, outputMd, mdStatement.Output)
		require.Equal(t, storyMd, mdStatement.Story)
		require.Equal(t, scoringMd, mdStatement.Scoring)
		require.Empty(t, mdStatement.Notes)
	}

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	parsedMdStatements2 := storedTask.MarkdownStatements
	require.Equal(t, 1, len(parsedMdStatements2))
	for _, mdStatement := range parsedMdStatements2 {
		lang := mdStatement.Language
		require.NotNil(t, lang)
		require.Equal(t, "lv", lang)
		require.Equal(t, inputMd, mdStatement.Input)
		require.Equal(t, outputMd, mdStatement.Output)
		require.Equal(t, storyMd, mdStatement.Story)
		require.Equal(t, scoringMd, mdStatement.Scoring)
		require.Empty(t, mdStatement.Notes)
	}
}
