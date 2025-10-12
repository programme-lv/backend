package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
)

// ExportTask exports a task as a ZIP file and streams it directly to the client.
func (h *taskHttpHandler) ExportTask(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	taskId := chi.URLParam(r, "taskId")

	l := h.logger(r.Context()).With(
		"handler", "ExportTask",
		"task_id", taskId,
	)

	// if client is already gone, stop immediately to avoid doing any work.
	select {
	case <-r.Context().Done():
		l.Warn("client disconnected before export started")
		return
	default:
	}

	// Call service to get ZIP bytes
	zipBytes, exportTaskAsZipErr := h.taskSrvc.ExportTaskAsZip(r.Context(), taskId)
	if exportTaskAsZipErr != nil {
		jsonresp.FromError(w, exportTaskAsZipErr)
		return
	}

	duration := time.Since(startTime)
	l.Info("task export completed successfully",
		"zip_size_bytes", len(zipBytes),
		"duration_ms", duration.Milliseconds(),
	)

	// Set response headers for file download
	filename := fmt.Sprintf("%s.zip", taskId)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	// Stream ZIP data directly to client
	_, err := w.Write(zipBytes)
	if err != nil {
		l.Warn("write ZIP to response", "error", err)
		return
	}
}
