package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingPDFStatement(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklV2dot5Path)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	expectedPdfPath := filepath.Join(kvadrputeklV2dot5Path, "statements", "pdf", "lv.pdf")
	expectedPdf, err := os.ReadFile(expectedPdfPath)
	require.NoErrorf(t, err, "failed to read PDF file: %v", err)

	actualPdf, err := parsedTask.GetPDFStatement("lv")
	require.NoErrorf(t, err, "failed to get PDF statement: %v", err)

	assert.Equal(t, expectedPdf, actualPdf)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	actualPdf2, err := storedTask.GetPDFStatement("lv")
	require.NoErrorf(t, err, "failed to get PDF statement: %v", err)
	assert.Equal(t, expectedPdf, actualPdf2)
}
