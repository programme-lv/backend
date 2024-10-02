package fstask

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type ArchiveFile struct {
	RelativePath string
	Content      []byte
}

func (dir TaskDir) ReadArchiveFiles() (res []ArchiveFile, err error) {
	requiredSpec := SemVer{major: 2, minor: 5}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		err = fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
		return
	}

	res = make([]ArchiveFile, 0)
	if _, err = os.Stat(filepath.Join(dir.Path, "archive")); os.IsNotExist(err) {
		err = nil
		return
	}

	archivePath := filepath.Join(dir.Path, "archive")
	err = filepath.Walk(archivePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(archivePath, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		res = append(res, ArchiveFile{
			RelativePath: relativePath,
			Content:      content,
		})
		return nil
	})

	return
}

func (task *Task) LoadArchiveFilesFromDir(dir TaskDir) error {
	archiveFiles, err := dir.ReadArchiveFiles()
	if err != nil {
		return fmt.Errorf("failed to read archive files: %w", err)
	}
	task.ArchiveFiles = archiveFiles
	return nil
}
