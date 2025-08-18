package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
)

// ExportTask exports a task as a ZIP file and streams it directly to the client.
func (h *taskHttpHandler) ExportTask(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	taskId := chi.URLParam(r, "taskId")

	logger := h.logger(r.Context()).With(
		"handler", "ExportTask",
		"task_id", taskId,
	)

	logger.Info("starting task export")

	// serialize access
	h.exportMu.Lock()
	defer h.exportMu.Unlock()

	// If client is already gone, stop immediately to avoid doing any work.
	select {
	case <-r.Context().Done():
		logger.Warn("client disconnected before export started")
		return
	default:
	}

	// Call service to get ZIP bytes
	zipBytes, err := h.taskSrvc.ExportTaskAsZip(r.Context(), taskId)
	if err != nil {
		logger.Error("failed to export task", "error", err)
		httpjson.HandleSrvcError(logger, w, err)
		return
	}

	duration := time.Since(startTime)
	logger.Info("task export completed successfully",
		"zip_size_bytes", len(zipBytes),
		"duration_ms", duration.Milliseconds(),
	)

	// Set response headers for file download
	filename := fmt.Sprintf("%s.zip", taskId)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	// Stream ZIP data directly to client
	_, err = w.Write(zipBytes)
	if err != nil {
		logger.Error("failed to write ZIP to response", "error", err)
		return
	}

	logger.Info("ZIP file streamed to client successfully")
}
