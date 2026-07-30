package taskzip

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	points := uint32(40)
	score := uint32(40)
	task := Task{
		Version: 1,
		ID:      "sum",
		Name:    map[string]string{"en": "Sum"},
		Testing: Testing{Type: "checker", CPUMs: 1000, MemMiB: 256},
		Readme:  []byte("maintainer notes\n"),
		Statements: map[string][]byte{
			"en": []byte("Add the numbers.\n"),
		},
		StatementImages: map[string][]byte{"plot.png": []byte("PNG")},
		Examples: []Example{{
			Input: []byte("1 2\n"), Output: []byte("3\n"),
			Notes: map[string][]byte{"en": []byte("Addition.\n"), "lv": []byte("Saskaitīšana.\n")},
		}},
		Tests:      []Test{{Input: []byte("1 2\n"), Output: []byte("3\n")}},
		Checker:    []byte("int main() {}\n"),
		Subtasks:   []Subtask{{Points: &points, Tests: "001-001", Description: map[string]string{"en": "All."}}},
		Solutions:  []Solution{{Filename: "full.cpp", Subtasks: []uint32{1}, Score: &score, Data: []byte("int main() {}\n")}},
		Origin:     &Origin{Olymp: "LIO", Divisions: []string{"junior", "senior"}},
		Extensions: map[string]any{"site": map[string]any{"key": "value"}},
	}

	first, err := Write(task)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(first)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != task.ID || len(got.Origin.Divisions) != 2 ||
		string(got.Examples[0].Notes["lv"]) != "Saskaitīšana.\n" {
		t.Fatalf("unexpected round trip: %#v", got)
	}
	second, err := Write(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Write is not deterministic")
	}
}

func TestGroupedScoring(t *testing.T) {
	task := minimalTask()
	task.Tests = append(task.Tests, Test{Input: []byte("2\n"), Output: []byte("2\n")})
	task.Subtasks = []Subtask{{Groups: "01-02"}}
	task.TestGroups = []TestGroup{
		{ID: 1, First: 1, Last: 1, Points: 30, Public: true},
		{ID: 2, First: 2, Last: 2, Points: 70},
	}
	data, err := Write(task)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TestGroups) != 2 || !got.TestGroups[0].Public {
		t.Fatalf("unexpected groups: %#v", got.TestGroups)
	}
}

func TestWrapperAndIgnoredDirectories(t *testing.T) {
	data, err := Write(minimalTask())
	if err != nil {
		t.Fatal(err)
	}
	files := unzipForTest(t, data)
	wrapped := map[string][]byte{
		"sum/task.toml":          files["task.toml"],
		"sum/statement/en.md":    files["statement/en.md"],
		"sum/tests/001i.txt":     files["tests/001i.txt"],
		"sum/tests/001o.txt":     files["tests/001o.txt"],
		"sum/archive/source.pdf": []byte("ignored"),
		"sum/testspec/tests.txt": []byte("ignored"),
	}
	if _, err := Read(zipForTest(t, wrapped)); err != nil {
		t.Fatal(err)
	}
	wrapped["other/file"] = []byte("bad")
	if _, err := Read(zipForTest(t, wrapped)); err == nil {
		t.Fatal("accepted multiple wrappers")
	}
}

func TestClassifiedUnsupportedFeatures(t *testing.T) {
	task := minimalTask()
	task.Testing.Type = "interactor"
	if _, err := Write(task); !errors.Is(err, ErrInteractive) {
		t.Fatalf("got %v", err)
	}

	files := archiveFiles(t, minimalTask())
	files["attached/grader.h"] = []byte("x")
	if _, err := Read(zipForTest(t, files)); !errors.Is(err, ErrAttached) {
		t.Fatalf("got %v", err)
	}
}

func TestRejectsUnsafeDuplicateUnknownAndStrictTOML(t *testing.T) {
	files := archiveFiles(t, minimalTask())
	files["unknown.txt"] = []byte("x")
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted unknown path")
	}

	files = archiveFiles(t, minimalTask())
	files["task.toml"] = append(files["task.toml"], []byte("\nunknown = 1\n")...)
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted unknown TOML field")
	}

	if _, err := Read(zipEntriesForTest(t, []zipEntry{
		{name: "../task.toml", data: []byte("x")},
	})); err == nil {
		t.Fatal("accepted traversal")
	}

	if _, err := Read(zipEntriesForTest(t, []zipEntry{
		{name: "task.toml", data: []byte("x")},
		{name: "task.toml", data: []byte("x")},
	})); err == nil {
		t.Fatal("accepted duplicate")
	}
}

func TestRejectsNonConsecutiveFilesAndRanges(t *testing.T) {
	files := archiveFiles(t, minimalTask())
	delete(files, "tests/001i.txt")
	files["tests/002i.txt"] = []byte("1\n")
	files["tests/002o.txt"] = []byte("1\n")
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted non-consecutive tests")
	}

	task := minimalTask()
	points := uint32(10)
	task.Subtasks = []Subtask{{Points: &points, Tests: "002-002"}}
	if _, err := Write(task); err == nil {
		t.Fatal("accepted out-of-order subtask range")
	}
}

func TestRejectsZeroIndicesAndOverflowingGroupPoints(t *testing.T) {
	files := archiveFiles(t, minimalTask())
	delete(files, "tests/001i.txt")
	files["tests/000i.txt"] = []byte("1\n")
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted zero test index")
	}

	task := minimalTask()
	task.Subtasks = []Subtask{{Groups: "01-01"}}
	task.TestGroups = []TestGroup{{ID: 1, First: 1, Last: 1, Points: 1}}
	files = archiveFiles(t, task)
	files["tgroups.txt"] = []byte("01: 001-001 4294967296p\n")
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted overflowing group points")
	}
}

func TestRejectsMissingDifficultyAndWrappedOversizedMetadata(t *testing.T) {
	task := minimalTask()
	task.Metadata = &Metadata{Topics: []string{"graphs"}}
	if _, err := Write(task); err == nil {
		t.Fatal("accepted metadata without difficulty")
	}

	files := map[string][]byte{
		"sum/task.toml":       make([]byte, (1<<20)+1),
		"sum/statement/en.md": []byte("Statement.\n"),
		"sum/tests/001i.txt":  []byte("1\n"),
		"sum/tests/001o.txt":  []byte("1\n"),
	}
	if _, err := Read(zipForTest(t, files)); err == nil {
		t.Fatal("accepted oversized wrapped task.toml")
	}
}

func TestRejectsTooManyEntries(t *testing.T) {
	entries := make([]zipEntry, maxImportFiles+1)
	for i := range entries {
		entries[i] = zipEntry{name: fmt.Sprintf("archive/%05d", i)}
	}
	if _, err := Read(zipEntriesForTest(t, entries)); err == nil {
		t.Fatal("accepted too many entries")
	}
}

func minimalTask() Task {
	return Task{
		Version:    1,
		ID:         "sum",
		Name:       map[string]string{"en": "Sum"},
		Testing:    Testing{Type: "simple", CPUMs: 1000, MemMiB: 256},
		Statements: map[string][]byte{"en": []byte("Statement.\n")},
		Tests:      []Test{{Input: []byte("1\n"), Output: []byte("1\n")}},
	}
}

func archiveFiles(t *testing.T, task Task) map[string][]byte {
	t.Helper()
	data, err := Write(task)
	if err != nil {
		t.Fatal(err)
	}
	return unzipForTest(t, data)
}

func unzipForTest(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		_, _ = out.ReadFrom(r)
		_ = r.Close()
		files[f.Name] = out.Bytes()
	}
	return files
}

func zipForTest(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	entries := make([]zipEntry, 0, len(files))
	for name, data := range files {
		entries = append(entries, zipEntry{name: name, data: data})
	}
	return zipEntriesForTest(t, entries)
}

type zipEntry struct {
	name string
	data []byte
}

func zipEntriesForTest(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, entry := range entries {
		w, err := zw.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(fmt.Errorf("close zip: %w", err))
	}
	return out.Bytes()
}
