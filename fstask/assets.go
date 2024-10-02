package fstask

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type AssetFile struct {
	RelativePath string
	Content      []byte
}

func (dir TaskDir) ReadAssetFiles() (res []AssetFile, err error) {
	res = make([]AssetFile, 0)

	assetsPath := filepath.Join(dir.Path, "assets")

	if _, statErr := os.Stat(assetsPath); os.IsNotExist(statErr) {
		err = nil
		return
	} else if statErr != nil {
		err = fmt.Errorf("error accessing assets directory: %w", statErr)
		return
	}

	err = filepath.WalkDir(assetsPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == assetsPath {
			return nil
		}

		if d.IsDir() {
			return nil // Continue walking through directories.
		}

		relativePath, relErr := filepath.Rel(assetsPath, path)
		if relErr != nil {
			return relErr
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("error reading asset '%s': %w", relativePath, readErr)
		}

		res = append(res, AssetFile{
			RelativePath: relativePath,
			Content:      content,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking through assets directory: %w", err)
	}

	return
}

// LoadAssetsFromDir loads asset files into the Task from the specified TaskDir.
// It updates the Task's Assets field with the loaded assets.
func (task *Task) LoadAssetsFromDir(dir TaskDir) error {
	assetFiles, err := dir.ReadAssetFiles()
	if err != nil {
		return fmt.Errorf("failed to read asset files: %w", err)
	}
	task.Assets = assetFiles
	return nil
}
