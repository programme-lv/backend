package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
	"github.com/programme-lv/backend/task/srvc"
)

type PutStatementRequest struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes"`
	Scoring string `json:"scoring"`
	Talk    string `json:"talk"`
	Example string `json:"example"`
}

func (h *taskHttpHandler) PutStatement(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	langIso639 := chi.URLParam(r, "langIso639")

	if langIso639 != "lv" {
		http.Error(w, "only lv is supported", http.StatusBadRequest)
		return
	}

	var req PutStatementRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.taskSrvc.UpdateStatementMd(r.Context(), taskId, srvc.MarkdownStatement{
		LangIso639: langIso639,
		Story:      req.Story,
		Input:      req.Input,
		Output:     req.Output,
		Notes:      req.Notes,
		Scoring:    req.Scoring,
		Talk:       req.Talk,
		Example:    req.Example,
	})
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	err = httpjson.Success(w, nil)
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}

func (h *taskHttpHandler) DeleteStatementImage(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	filename := chi.URLParam(r, "filename")

	// URL decode the filename parameter as it may contain special characters
	decodedFilename, err := url.QueryUnescape(filename)
	if err != nil {
		errMsg := fmt.Sprintf("failed to decode filename: %v", err)
		errCode := "invalid_filename"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	err = h.taskSrvc.DeleteStatementImage(r.Context(), taskId, decodedFilename)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = httpjson.Success(w, map[string]string{"status": "ok"})
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}

func (h *taskHttpHandler) UploadStatementImage(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	err := r.ParseMultipartForm(10 << 20) // max 10MB
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse multipart form (maybe the image is too large?): %v", err)
		errCode := "failed_to_parse_multipart_form"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// get image as byte array
	image, header, err := r.FormFile("image")
	if err != nil {
		errMsg := fmt.Sprintf("failed to get image: %v", err)
		errCode := "failed_to_get_image"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	defer image.Close()

	uploadedFilename := header.Filename
	// let's make sure the filename is clean and safe. allow only alphanumeric characters, underscores, and hyphens
	filenameWithoutExt := strings.TrimSuffix(uploadedFilename, filepath.Ext(uploadedFilename))
	imageFilenameExt := filepath.Ext(uploadedFilename)
	// otherwise throw bad request with a list of allowed characters
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !allowedChars.MatchString(filenameWithoutExt) {
		errMsg := fmt.Sprintf("invalid filename (only alphanumeric characters, underscores, and hyphens are allowed): %s", uploadedFilename)
		errCode := "invalid_filename"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	if len(filenameWithoutExt) > 100 {
		errMsg := fmt.Sprintf("filename is too long (max 100 characters): %s", filenameWithoutExt)
		errCode := "filename_too_long"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	cantContain := []string{"CON", "PRN", "AUX", "NUL", "COM", "LPT"}
	if len(filenameWithoutExt) < 4 && slices.Contains(cantContain, filenameWithoutExt) {
		errMsg := fmt.Sprintf("invalid filename (may contain reserved filenames): %s", filenameWithoutExt)
		errCode := "invalid_filename"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// get specified and detected MIME types
	_, imageMimeType, err := getUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get MIME types: %v", err)
		errCode := "failed_to_get_mimes"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// can we somehow check that imageFilenameExt matches the mime type?
	if !isExtensionValidForMIME(imageFilenameExt, imageMimeType) {
		errMsg := fmt.Sprintf("file extension '%s' does not match detected MIME type '%s'", imageFilenameExt, imageMimeType)
		errCode := "invalid_file_extension"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read image: %v", err)
		errCode := "failed_to_read_image"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	uri, err := h.taskSrvc.UploadStatementImage(r.Context(), taskId, uploadedFilename, imageMimeType, imageBytes)
	if err != nil {
		writeSrvcError(w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = httpjson.Success(w, uri)
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}
