package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

// packageDirNames lists top-level directories that never belong to the stored
// task. The backend ignores testspec/, .taskzip/ holds locally
// generated caches, and .git/ plus OS metadata fail its ZIP path validation.
// archive/ is packaged; the backend stores those leftover files.
// Skipping them keeps uploads small and avoids predictable server errors.
var packageDirNames = map[string]bool{
	".git":     true,
	".taskzip": true,
	"testspec": true,
	"target":   true,
	"__MACOSX": true,
}

// readTaskPackage returns the ZIP bytes for a TaskZip directory or .zip file
// together with the archive task ID. It validates the package with the same
// reader the backend uses, so invalid packages fail before any upload.
func readTaskPackage(path string) ([]byte, string, error) {
	return readTaskPackageMode(path, false)
}

func readTaskPackageMode(path string, partial bool) ([]byte, string, error) {
	zipBytes, task, err := readTaskPackageWithTaskMode(path, partial)
	if err != nil {
		return nil, "", err
	}
	return zipBytes, task.ID, nil
}

func readTaskPackageWithTask(path string) ([]byte, taskzipv1.Task, error) {
	return readTaskPackageWithTaskMode(path, false)
}

func readTaskPackageWithTaskMode(path string, partial bool) ([]byte, taskzipv1.Task, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, taskzipv1.Task{}, fmt.Errorf("stat %s: %w", path, err)
	}

	var zipBytes []byte
	if info.IsDir() {
		zipBytes, err = zipTaskDir(path)
		if err != nil {
			return nil, taskzipv1.Task{}, err
		}
	} else {
		zipBytes, err = os.ReadFile(path)
		if err != nil {
			return nil, taskzipv1.Task{}, fmt.Errorf("read %s: %w", path, err)
		}
	}

	var task taskzipv1.Task
	if partial {
		task, err = taskzipv1.ReadPatch(zipBytes)
	} else {
		task, err = taskzipv1.Read(zipBytes)
	}
	if err != nil {
		return nil, taskzipv1.Task{}, fmt.Errorf("invalid TaskZip package %s: %w", path, err)
	}
	return zipBytes, task, nil
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

// canonicalTask normalizes fields the backend cannot represent in exported
// TaskZip files, so local and remote hashes describe the same content.
func canonicalTask(task taskzipv1.Task) taskzipv1.Task {
	if len(task.Readme) == 0 {
		task.Readme = nil
	}
	if len(task.Checker) == 0 {
		task.Checker = nil
	}
	if len(task.Interactor) == 0 {
		task.Interactor = nil
	}
	for i := range task.Solutions {
		task.Solutions[i].Score = nil
	}
	return task
}

// taskSectionDigests hashes the canonical TaskZip files for each update section.
func taskSectionDigests(task taskzipv1.Task) (map[string]string, map[string]string, int, uint64, error) {
	files, err := taskzipv1.FileMap(canonicalTask(task))
	if err != nil {
		return nil, nil, 0, 0, err
	}
	hashes := map[string]string{}
	for name, data := range files {
		if isManagedPath(name) {
			hashes[name] = sha256Hex(data)
		}
	}
	out := map[string]string{
		"task":       digestTask(files),
		"statement":  digestSection(files, "statement/"),
		"examples":   digestSection(files, "examples/"),
		"solutions":  digestSection(files, "solutions/"),
		"tests":      digestSection(files, "tests/"),
		"readme":     digestOptional(files, "readme.md"),
		"checker":    digestOptional(files, "checker.cpp"),
		"interactor": digestOptional(files, "interactor.cpp"),
	}
	var size uint64
	for _, test := range task.Tests {
		size += uint64(len(test.Input) + len(test.Output))
	}
	return out, hashes, len(task.Tests) * 2, size, nil
}

func isManagedPath(name string) bool {
	if name == "task.toml" || name == "tgroups.txt" {
		return true
	}
	return sectionForPath(name) != ""
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func digestSection(files map[string][]byte, prefix string) string {
	var keys []string
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			keys = append(keys, name)
		}
	}
	return digestFiles(files, keys)
}

func digestTask(files map[string][]byte) string {
	var keys []string
	for _, name := range []string{"task.toml", "tgroups.txt"} {
		if _, ok := files[name]; ok {
			keys = append(keys, name)
		}
	}
	return digestFiles(files, keys)
}

func digestOptional(files map[string][]byte, name string) string {
	if _, ok := files[name]; !ok {
		return digestFiles(files, nil)
	}
	return digestFiles(files, []string{name})
}

func digestFiles(files map[string][]byte, keys []string) string {
	sort.Strings(keys)
	h := sha256.New()
	for _, name := range keys {
		sum := sha256.Sum256(files[name])
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write([]byte(hex.EncodeToString(sum[:])))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sectionForPath(name string) string {
	switch {
	case strings.HasPrefix(name, "statement/"):
		return "statement"
	case strings.HasPrefix(name, "examples/"):
		return "examples"
	case strings.HasPrefix(name, "solutions/"):
		return "solutions"
	case strings.HasPrefix(name, "tests/"):
		return "tests"
	case name == "readme.md":
		return "readme"
	case name == "checker.cpp":
		return "checker"
	case name == "interactor.cpp":
		return "interactor"
	default:
		return ""
	}
}

// buildPatchZip writes task.toml and only the changed sections.
func buildPatchZip(task taskzipv1.Task, sections map[string]bool) ([]byte, error) {
	files, err := taskzipv1.FileMap(canonicalTask(task))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		section := sectionForPath(name)
		keep := name == "task.toml" || name == "tgroups.txt"
		keep = keep || (section != "" && sections[section])
		if keep {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
		if _, err := writer.Write(files[name]); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close patch zip: %w", err)
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
