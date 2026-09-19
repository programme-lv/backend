package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const minimalTaskTOML = `taskzip = 1
id = "addtwo"
name = { en = "Add Two" }

[testing]
type = "simple"
cpu_ms = 1000
mem_mib = 256
`

// writeMinimalTask creates the smallest valid TaskZip package under root.
func writeMinimalTask(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"task.toml":       minimalTaskTOML,
		"tests/001i.txt":  "1 2\n",
		"tests/001o.txt":  "3\n",
		"statement/en.md": "Add two numbers.\n",
	}
	for name, content := range files {
		writeFile(t, root, name, content)
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func zipNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		names[file.Name] = true
	}
	return names
}

func TestReadTaskPackageFromDirSkipsNonTaskPaths(t *testing.T) {
	root := t.TempDir()
	writeMinimalTask(t, root)

	noise := map[string]string{
		".taskzip/generated/001i.txt": "cached\n",
		"archive/old.txt":             "old\n",
		"testspec/generator.cpp":      "int main() {}\n",
		".git/config":                 "[core]\n",
		".DS_Store":                   "junk",
		"__MACOSX/._task.toml":        "junk",
	}
	for name, content := range noise {
		writeFile(t, root, name, content)
	}

	data, id, err := readTaskPackage(root)
	if err != nil {
		t.Fatalf("readTaskPackage: %v", err)
	}
	if id != "addtwo" {
		t.Fatalf("id = %q, want addtwo", id)
	}

	names := zipNames(t, data)
	for _, want := range []string{"task.toml", "tests/001i.txt", "tests/001o.txt", "statement/en.md"} {
		if !names[want] {
			t.Errorf("archive is missing %s", want)
		}
	}
	for name := range names {
		if _, skipped := noise[name]; skipped {
			t.Errorf("archive contains skipped path %s", name)
		}
	}
}

func TestReadTaskPackageFromZip(t *testing.T) {
	root := t.TempDir()
	writeMinimalTask(t, root)
	data, _, err := readTaskPackage(root)
	if err != nil {
		t.Fatalf("readTaskPackage(dir): %v", err)
	}

	zipPath := filepath.Join(t.TempDir(), "addtwo.zip")
	if err := os.WriteFile(zipPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, id, err := readTaskPackage(zipPath)
	if err != nil {
		t.Fatalf("readTaskPackage(zip): %v", err)
	}
	if id != "addtwo" {
		t.Fatalf("id = %q, want addtwo", id)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("zip bytes changed between reads")
	}
}

func TestReadTaskPackageAcceptsWrapperDirectory(t *testing.T) {
	parent := t.TempDir()
	writeMinimalTask(t, filepath.Join(parent, "addtwo"))

	data, id, err := readTaskPackage(parent)
	if err != nil {
		t.Fatalf("readTaskPackage: %v", err)
	}
	if id != "addtwo" {
		t.Fatalf("id = %q, want addtwo", id)
	}
	if names := zipNames(t, data); !names["addtwo/task.toml"] {
		t.Fatalf("archive = %v, want addtwo/task.toml", names)
	}
}

func TestReadTaskPackageRejectsInvalid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "readme.md", "not a task\n")
	if _, _, err := readTaskPackage(root); err == nil {
		t.Fatal("readTaskPackage accepted a directory without task.toml")
	}
}
