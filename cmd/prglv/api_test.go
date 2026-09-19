package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadTaskSendsMultipartForm(t *testing.T) {
	var gotAuth, gotFilename string
	var gotBytes []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/upload" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		file, header, err := r.FormFile("task_zip")
		if err != nil {
			t.Errorf("form file: %v", err)
			return
		}
		defer file.Close()
		gotFilename = header.Filename
		gotBytes, _ = io.ReadAll(file)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]string{"task_id": "addtwo"},
		})
	}))
	defer server.Close()

	id, err := uploadTask(context.Background(), server.URL, "secret", []byte("zip-bytes"), "")
	if err != nil {
		t.Fatalf("uploadTask: %v", err)
	}
	if id != "addtwo" {
		t.Fatalf("id = %q, want addtwo", id)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotFilename != "task.zip" {
		t.Fatalf("filename = %q", gotFilename)
	}
	if string(gotBytes) != "zip-bytes" {
		t.Fatalf("field bytes = %q", gotBytes)
	}
}

func TestUploadTaskSendsOverrideID(t *testing.T) {
	var gotOverride string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOverride = r.URL.Query().Get("override_id")
		_ = r.ParseMultipartForm(1 << 20)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]string{"task_id": "custom"},
		})
	}))
	defer server.Close()

	id, err := uploadTask(context.Background(), server.URL, "k", []byte("x"), "custom")
	if err != nil {
		t.Fatalf("uploadTask: %v", err)
	}
	if id != "custom" {
		t.Fatalf("id = %q, want custom", id)
	}
	if gotOverride != "custom" {
		t.Fatalf("override_id = %q", gotOverride)
	}
}

func TestUploadTaskReportsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"code":    "task_already_exists",
			"message": "uzdevums jau eksistē",
		})
	}))
	defer server.Close()

	_, err := uploadTask(context.Background(), server.URL, "k", []byte("x"), "")
	if err == nil {
		t.Fatal("uploadTask accepted an error response")
	}
	if !strings.Contains(err.Error(), "jau eksistē") || !strings.Contains(err.Error(), "task_already_exists") {
		t.Fatalf("err = %v", err)
	}
}

func TestExportTaskReturnsArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/addtwo/export" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("zip-bytes"))
	}))
	defer server.Close()

	data, err := exportTask(context.Background(), server.URL, "k", "addtwo")
	if err != nil {
		t.Fatalf("exportTask: %v", err)
	}
	if string(data) != "zip-bytes" {
		t.Fatalf("data = %q", data)
	}
}

func TestRunExportWritesFileAndRefusesOverwrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("zip-bytes"))
	}))
	defer server.Close()

	out := filepath.Join(t.TempDir(), "addtwo.zip")
	args := []string{"--api-url", server.URL, "--api-key", "k", "--out", out, "addtwo"}
	if err := runExport(args); err != nil {
		t.Fatalf("runExport: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "zip-bytes" {
		t.Fatalf("file = %q", data)
	}

	if err := runExport(args); err == nil {
		t.Fatal("runExport overwrote an existing file without --force")
	}
}
