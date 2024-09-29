package fstask_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/programme-lv/backend/fstask"
	"github.com/stretchr/testify/require"
)

func TestReadingWritingIllustrationImage(t *testing.T) {
	parsedTask, err := fstask.Read(kvadrputeklPath)
	require.NoErrorf(t, err, "failed to read task: %v", err)

	// read illustration image
	imgPath := filepath.Join(kvadrputeklPath, "assets", "illustration.png")
	imgAsset2, err := os.ReadFile(imgPath)
	require.NoErrorf(t, err, "failed to read illustration image: %v", err)

	parsedImg := parsedTask.GetIllustrationImage()
	require.NotNil(t, parsedImg)
	expectedImgAsset := &fstask.Asset{
		RelativePath: "illustration.png",
		Content:      imgAsset2,
	}
	require.Equal(t, len(parsedTask.GetAssets()), 3)
	require.Equal(t, expectedImgAsset.Content, parsedImg.Content)
	require.Equal(t, expectedImgAsset.RelativePath, parsedImg.RelativePath)
	require.Equal(t, expectedImgAsset, parsedImg)

	tmpDirectory, err := os.MkdirTemp("", "fstaskparser-test-")
	require.NoErrorf(t, err, "failed to create temporary directory: %v", err)
	defer os.RemoveAll(tmpDirectory)

	outputDirectory := filepath.Join(tmpDirectory, "kvadrputekl")
	t.Logf("Created directory for output: %s", outputDirectory)

	err = parsedTask.Store(outputDirectory)
	require.NoErrorf(t, err, "failed to store task: %v", err)

	storedTask, err := fstask.Read(outputDirectory)
	require.NoErrorf(t, err, "failed to read task: %v", err)
	parsedImgAsset2 := storedTask.GetIllustrationImage()
	require.NotNil(t, parsedImgAsset2)
	require.Equal(t, expectedImgAsset, parsedImgAsset2)
}
