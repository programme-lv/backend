package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

// packageDirNames lists top-level directories that never belong to the stored
// task. The backend ignores archive/ and testspec/, .taskzip/ holds locally
// generated caches, and .git/ plus OS metadata fail its ZIP path validation.
// Skipping them keeps uploads small and avoids predictable server errors.
var packageDirNames = map[string]bool{
	".git":     true,
	".taskzip": true,
	"archive":  true,
	"testspec": true,
	"target":   true,
	"__MACOSX": true,
}

// readTaskPackage returns the ZIP bytes for a TaskZip directory or .zip file
// together with the archive task ID. It validates the package with the same
// reader the backend uses, so invalid packages fail before any upload.
func readTaskPackage(path string) ([]byte, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("stat %s: %w", path, err)
	}

	var zipBytes []byte
	if info.IsDir() {
		zipBytes, err = zipTaskDir(path)
		if err != nil {
			return nil, "", err
		}
	} else {
		zipBytes, err = os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", path, err)
		}
	}

	task, err := taskzipv1.Read(zipBytes)
	if err != nil {
		return nil, "", fmt.Errorf("invalid TaskZip package %s: %w", path, err)
	}
	return zipBytes, task.ID, nil
}

// zipTaskDir archives the contents of root as a flat TaskZip ZIP.
// The backend accepts either a flat archive or one wrapper directory whose
// name matches the task ID, so a plain relative archive works for both a
// task directory and a parent directory that contains one.
func zipTaskDir(root string) ([]byte, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", root, err)
	}
	root = filepath.Clean(resolved)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		if entry.IsDir() {
			if packageDirNames[strings.SplitN(name, "/", 2)[0]] {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || entry.Name() == ".DS_Store" {
			return nil
		}
		return writeZipFile(zw, path, name)
	})
	if err != nil {
		_ = zw.Close()
		return nil, fmt.Errorf("package %s: %w", root, err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("package %s: %w", root, err)
	}
	return buf.Bytes(), nil
}

func writeZipFile(zw *zip.Writer, path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o644)
	header.SetModTime(info.ModTime())

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}
