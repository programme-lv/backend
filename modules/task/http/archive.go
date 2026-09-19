package http

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
)

type archiveFileJSON struct {
	Path      string `json:"path"`
	SzInBytes int    `json:"sz_in_bytes"`
}

// ListTaskArchive returns leftover archive/ files for {taskId}.
func (h *taskHttpHandler) ListTaskArchive(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	files, err := h.taskSrvc.ListArchiveFiles(r.Context(), taskId)
	if err != nil {
		jsonresp.WriteError(w, err)
		return
	}
	out := make([]archiveFileJSON, 0, len(files))
	for _, file := range files {
		out = append(out, archiveFileJSON{Path: file.Path, SzInBytes: file.SzInBytes})
	}
	_ = jsonresp.Success(w, out)
}

// DownloadTaskArchiveFile writes one leftover archive file.
func (h *taskHttpHandler) DownloadTaskArchiveFile(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	relPath := chi.URLParam(r, "*")
	data, filename, err := h.taskSrvc.DownloadArchiveFile(r.Context(), taskId, relPath)
	if err != nil {
		jsonresp.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(data)
}

// UploadTaskArchiveFile stores one leftover archive file.
func (h *taskHttpHandler) UploadTaskArchiveFile(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	if err := r.ParseMultipartForm(65 << 20); err != nil {
		jsonresp.BadRequest(w, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonresp.BadRequest(w, "read file")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (64<<20)+1))
	if err != nil {
		jsonresp.BadRequest(w, "read file")
		return
	}
	relPath := r.FormValue("path")
	if relPath == "" {
		relPath = path.Base(header.Filename)
	}
	if err := h.taskSrvc.UploadArchiveFile(r.Context(), taskId, relPath, body); err != nil {
		jsonresp.WriteError(w, err)
		return
	}
	_ = jsonresp.Success(w, struct{}{})
}

// DeleteTaskArchiveFile removes one leftover archive file.
func (h *taskHttpHandler) DeleteTaskArchiveFile(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	relPath := chi.URLParam(r, "*")
	if err := h.taskSrvc.DeleteArchiveFile(r.Context(), taskId, relPath); err != nil {
		jsonresp.WriteError(w, err)
		return
	}
	_ = jsonresp.Success(w, struct{}{})
}
