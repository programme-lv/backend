package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
)

// ExportTask writes the specified task into a hardcoded directory in a simple JSON form.
// For now we do not zip; this is a placeholder for using taskzip later.
// Concurrency: only one export request is processed at a time.
func (h *taskHttpHandler) ExportTask(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	// serialize access
	h.exportMu.Lock()
	defer h.exportMu.Unlock()

	// If client is already gone, stop immediately to avoid doing any work.
	select {
	case <-r.Context().Done():
		return
	default:
	}

	// fetch full task via narrow interface
	t, err := h.taskSrvc.GetTask(r.Context(), taskId)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	// export dir (configurable on handler; default set in constructor)
	baseDir := h.exportDir
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to create export dir: %v", err), http.StatusInternalServerError, "export_mkdir_failed")
		return
	}

	dstDir := filepath.Join(baseDir, t.ShortId)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to create task dir: %v", err), http.StatusInternalServerError, "export_taskdir_failed")
		return
	}

	// Write a simple JSON dump as a stub. Later this will use taskzip to structure properly.
	jsonPath := filepath.Join(dstDir, "task.json")
	f, err := os.Create(jsonPath)
	if err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to create json file: %v", err), http.StatusInternalServerError, "export_file_create_failed")
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(t); err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to encode task: %v", err), http.StatusInternalServerError, "export_encode_failed")
		return
	}

	_ = httpjson.Success(w, map[string]string{
		"status":   "ok",
		"exported": dstDir,
	})
}
