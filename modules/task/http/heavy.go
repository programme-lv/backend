package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	fname "github.com/programme-lv/backend/common/fname"
	"github.com/programme-lv/backend/common/img"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/mimetype"
	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/task/storage"
)

// UploadTaskResponse is the JSON body returned after importing a task ZIP.
type UploadTaskResponse struct {
	TaskId string `json:"task_id"`
}

// UploadTask imports a TaskZip v1 archive from multipart field task_zip
// and writes the created task ID as JSON.
// Query parameter override_id, if set, replaces the archive's short ID.
func (h *taskHttpHandler) UploadTask(w http.ResponseWriter, r *http.Request) {
	logger := h.logger(r.Context()).With("handler", "UploadTask")

	if err := r.ParseMultipartForm(512 << 20); err != nil { // 512MB cap
		msg := fmt.Sprintf("parse multipart form: %v", err)
		jsonresp.BadRequest(w, msg)
		return
	}

	file, _, err := r.FormFile("task_zip")
	if err != nil {
		msg := fmt.Sprintf("read task_zip: %v", err)
		jsonresp.BadRequest(w, msg)
		return
	}
	defer file.Close()

	zipBytes, err := io.ReadAll(file)
	if err != nil {
		msg := fmt.Sprintf("read all zip bytes: %v", err)
		logger.Error(msg, "error", err)
		jsonresp.BadRequest(w, msg)
		return
	}

	overrideId := r.URL.Query().Get("override_id")

	createdId, importTaskErr := h.taskSrvc.ImportTaskFromZip(r.Context(), zipBytes, overrideId)
	if importTaskErr != nil {
		jsonresp.WriteError(w, importTaskErr)
		return
	}

	_ = jsonresp.Success(w, UploadTaskResponse{TaskId: createdId})
}

// UpdateTask replaces task content from multipart field task_zip without
// deleting the task. Query parameter allow_test_changes=true permits official
// test or scoring changes and bumps the task's tests_version.
func (h *taskHttpHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		jsonresp.BadRequest(w, fmt.Sprintf("parse multipart form: %v", err))
		return
	}
	file, _, err := r.FormFile("task_zip")
	if err != nil {
		jsonresp.BadRequest(w, fmt.Sprintf("read task_zip: %v", err))
		return
	}
	defer file.Close()
	zipBytes, err := io.ReadAll(file)
	if err != nil {
		jsonresp.BadRequest(w, fmt.Sprintf("read all zip bytes: %v", err))
		return
	}
	var result srvc.UpdateResult
	var updateErr error
	if manifest := r.FormValue("manifest"); manifest != "" {
		var meta srvc.PatchMeta
		if err := json.Unmarshal([]byte(manifest), &meta); err != nil {
			jsonresp.BadRequest(w, fmt.Sprintf("parse manifest: %v", err))
			return
		}
		result, updateErr = h.taskSrvc.PatchTaskFromZip(r.Context(), taskID, zipBytes, meta)
	} else {
		allowTests := r.URL.Query().Get("allow_test_changes") == "true"
		result, updateErr = h.taskSrvc.UpdateTaskFromZip(r.Context(), taskID, zipBytes, allowTests)
	}
	if updateErr != nil {
		jsonresp.WriteError(w, updateErr)
		return
	}
	h.getTaskViewCache.Delete(taskID)
	h.getTaskListCache.Delete("")
	h.getTaskFiltersCache.Delete("")
	_ = jsonresp.Success(w, result)
}

// UploadStatementImage stores a statement image from multipart field image.
func (h *taskHttpHandler) UploadStatementImage(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	err := r.ParseMultipartForm(10 << 20) // max 10MB
	if err != nil {
		errMsg := fmt.Sprintf("parse multipart form (image may be too large): %v", err)
		jsonresp.BadRequest(w, errMsg)
		return
	}

	image, header, err := r.FormFile("image")
	if err != nil {
		msg := fmt.Sprintf("read image: %v", err)
		jsonresp.BadRequest(w, msg)
		return
	}
	defer image.Close()

	uploadedFilename := header.Filename
	_, imageFilenameExt, vErr := fname.ValidateUploadedImageFilename(uploadedFilename)
	if vErr != nil {
		code := "invalid_filename"
		if ve, ok := vErr.(*fname.FilenameValidationError); ok {
			code = ve.Code
		}
		jsonresp.WriteCustom(w, vErr.Error(), http.StatusBadRequest, code)
		return
	}

	_, imageMimeType, err := mimetype.GetUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("detect MIME type: %v", err)
		errCode := "failed_to_get_mimes"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	if !mimetype.IsExtensionValidForMIME(imageFilenameExt, imageMimeType) {
		msg := fmt.Sprintf("file ext '%s' does not match detected MIME type '%s'", imageFilenameExt, imageMimeType)
		jsonresp.BadRequest(w, msg)
		return
	}

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		msg := fmt.Sprintf("read image: %v", err)
		jsonresp.BadRequest(w, msg)
		return
	}

	uri, uploadImageErr := h.taskSrvc.UploadStatementImage(r.Context(), taskId, uploadedFilename, imageMimeType, imageBytes)
	if uploadImageErr != nil {
		jsonresp.WriteError(w, uploadImageErr)
		return
	}

	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")

	err = jsonresp.Success(w, uri)
	if err != nil {
		slog.Error("write success json", "error", err)
	}
}

// UploadIllustration stores the task list illustration from multipart field image.
// The image must be between 1×1 and 2000×2000 pixels.
func (h *taskHttpHandler) UploadIllustration(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	err := r.ParseMultipartForm(10 << 20) // max 10MB
	if err != nil {
		errMsg := fmt.Sprintf("parse multipart form (image may be too large): %v", err)
		errCode := "failed_to_parse_multipart_form"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	image, header, err := r.FormFile("image")
	if err != nil {
		errMsg := fmt.Sprintf("read image: %v", err)
		errCode := "failed_to_get_image"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	defer image.Close()

	_, imageMimeType, err := mimetype.GetUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("detect MIME type: %v", err)
		errCode := "failed_to_get_mimes"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	if !strings.HasPrefix(imageMimeType, "image/") {
		errMsg := fmt.Sprintf("uploaded file is not an image (detected MIME type: %s)", imageMimeType)
		errCode := "invalid_mime_type"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		errMsg := fmt.Sprintf("read image: %v", err)
		errCode := "failed_to_read_image"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	width, height, err := img.GetImageDimensions(imageBytes, imageMimeType)
	if err != nil {
		errMsg := fmt.Sprintf("image dimensions: %v", err)
		errCode := "failed_to_get_image_dimensions"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		errMsg := "image dimensions are inadequate (must be between 1x1 and 2000x2000 pixels)"
		errCode := "inadequate_image_dimensions"
		jsonresp.WriteCustom(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	objectKey, uploadIllustrationErr := h.taskSrvc.UploadIllustrationImg(r.Context(), taskId, imageMimeType, imageBytes)
	if uploadIllustrationErr != nil {
		jsonresp.WriteError(w, uploadIllustrationErr)
		return
	}

	sizeInBytes := len(imageBytes)
	illustrationImg := srvc.IllustrationImage{
		ObjectKey: objectKey,
		SHA256:    storage.Digest(imageBytes),
		Ext:       path.Ext(objectKey),
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: sizeInBytes,
	}

	updateIllustrationErr := h.taskSrvc.UpdateIllustrationImg(r.Context(), taskId, illustrationImg)
	if updateIllustrationErr != nil {
		jsonresp.WriteError(w, updateIllustrationErr)
		return
	}

	h.getTaskViewCache.Delete(taskId)
	h.getTaskListCache.Delete("")

	err = jsonresp.Success(w, map[string]string{"object_key": objectKey})
	if err != nil {
		slog.Error("write success json", "error", err)
	}
}

// ExportTask writes the task as a ZIP attachment named {taskId}.zip.
func (h *taskHttpHandler) ExportTask(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	taskId := chi.URLParam(r, "taskId")

	l := h.logger(r.Context()).With(
		"handler", "ExportTask",
		"task_id", taskId,
	)

	// Skip the zip if the client already hung up.
	select {
	case <-r.Context().Done():
		l.Warn("client disconnected before export started")
		return
	default:
	}

	includeTests := r.URL.Query().Get("tests") != "false" &&
		r.URL.Query().Get("include_tests") != "false"
	var zipBytes []byte
	var exportTaskAsZipErr error
	if includeTests {
		zipBytes, exportTaskAsZipErr = h.taskSrvc.ExportTaskAsZip(r.Context(), taskId)
	} else {
		zipBytes, exportTaskAsZipErr = h.taskSrvc.ExportTaskAsZipWithoutTests(r.Context(), taskId)
	}
	if exportTaskAsZipErr != nil {
		jsonresp.WriteError(w, exportTaskAsZipErr)
		return
	}

	duration := time.Since(startTime)
	l.Info("task export completed successfully",
		"zip_size_bytes", len(zipBytes),
		"duration_ms", duration.Milliseconds(),
	)

	filename := fmt.Sprintf("%s.zip", taskId)
	if !includeTests {
		filename = fmt.Sprintf("%s-no-tests.zip", taskId)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	_, err := w.Write(zipBytes)
	if err != nil {
		l.Warn("write ZIP to response", "error", err)
		return
	}
}
