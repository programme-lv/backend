package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunUpdateSkipsUnchangedTests(t *testing.T) {
	root := t.TempDir()
	writeMinimalTask(t, root)
	_, task, err := readTaskPackageWithTask(root)
	if err != nil {
		t.Fatal(err)
	}
	taskID := task.ID
	sections, _, _, _, err := taskSectionDigests(task)
	if err != nil {
		t.Fatal(err)
	}

	var gotTests bool
	var gotManifest map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/"+taskID+"/manifest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"task_id": taskID, "tests_version": 1,
					"tests_sha256": sections["tests"], "tests_count": 1,
					"content_etag": "etag", "revision": 4, "sections": sections,
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/tasks/"+taskID:
			_ = r.ParseMultipartForm(1 << 20)
			file, _, err := r.FormFile("task_zip")
			if err != nil {
				t.Errorf("form file: %v", err)
				return
			}
			body, _ := io.ReadAll(file)
			file.Close()
			gotTests = zipNames(t, body)["tests/001i.txt"]
			_ = json.Unmarshal([]byte(r.FormValue("manifest")), &gotManifest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"task_id": taskID, "tests_changed": false, "tests_version": 1,
				},
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err = runUpdate([]string{
		"--api-url", server.URL, "--api-key", "k", "--yes", root,
	})
	if err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if gotTests {
		t.Fatal("sparse patch still contains tests")
	}
	base, _ := gotManifest["base_content_etag"].(string)
	if base != "etag" {
		t.Fatalf("base_content_etag = %q", base)
	}
	if rev, _ := gotManifest["base_revision"].(float64); rev != 4 {
		t.Fatalf("base_revision = %v", gotManifest["base_revision"])
	}
	sent, _ := gotManifest["sections"].([]any)
	if len(sent) != 0 {
		t.Fatalf("sections = %v, want none", sent)
	}
}

func TestDiffFiles(t *testing.T) {
	remote := map[string]string{
		"statement/en.md": "old",
		"tests/001i.txt":  "same",
		"gone.txt":        "gone",
	}
	local := map[string]string{
		"statement/en.md": "new",
		"tests/001i.txt":  "same",
		"added.txt":       "added",
	}
	added, modified, deleted := diffFiles(remote, local)
	if len(added) != 1 || added[0] != "added.txt" {
		t.Fatalf("added = %v", added)
	}
	if len(modified) != 1 || modified[0] != "statement/en.md" {
		t.Fatalf("modified = %v", modified)
	}
	if len(deleted) != 1 || deleted[0] != "gone.txt" {
		t.Fatalf("deleted = %v", deleted)
	}
}
