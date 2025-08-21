package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/programme-lv/backend/common/httpjson"
)

type UploadTaskResponse struct {
	TaskId string `json:"task_id"`
}

// UploadTask handles task upload via multipart/form-data with field name "task_zip".
// It calls the service to import the taskfs ZIP, and returns the created task id.
// Optional query parameter ?override_id=<new_id> can be used to override the task's short ID.
func (h *taskHttpHandler) UploadTask(w http.ResponseWriter, r *http.Request) {
	logger := h.logger(r.Context()).With("handler", "UploadTask")

	if err := r.ParseMultipartForm(512 << 20); err != nil { // 512MB cap
		httpjson.Error(w, fmt.Sprintf("failed to parse multipart form: %v", err), http.StatusBadRequest, "failed_to_parse_multipart_form")
		return
	}

	file, _, err := r.FormFile("task_zip")
	if err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to read task_zip: %v", err), http.StatusBadRequest, "missing_task_zip")
		return
	}
	defer file.Close()

	zipBytes, err := io.ReadAll(file)
	if err != nil {
		httpjson.Error(w, fmt.Sprintf("failed to read zip bytes: %v", err), http.StatusBadRequest, "failed_to_read_zip")
		return
	}

	// Check for optional ID override
	overrideId := r.URL.Query().Get("override_id")

	createdId, err := h.taskSrvc.ImportTaskFromZipWithId(r.Context(), zipBytes, overrideId)
	if err != nil {
		httpjson.HandleSrvcError(logger, w, err)
		return
	}

	_ = httpjson.Success(w, UploadTaskResponse{TaskId: createdId})
}
