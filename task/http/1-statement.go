package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	fname "github.com/programme-lv/backend/common/fname"
	"github.com/programme-lv/backend/common/httpjson"
	"github.com/programme-lv/backend/common/mimetype"
	"github.com/programme-lv/backend/task/srvc"
)

type PutStatementReq struct {
	Story   string `json:"story"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Notes   string `json:"notes"`
	Scoring string `json:"scoring"`
	Talk    string `json:"talk"`
	Example string `json:"example"`
}

func (h *taskHttpHandler) PutStatement(ctx context.Context,
	req PutStatementReq, params map[string]string,
) (response Empty, err error) {
	taskId := params["taskId"]
	lang := params["langIso639"]

	err = h.taskSrvc.UpdateStatementMd(ctx, taskId, srvc.MarkdownStatement{
		LangIso639: lang,
		Story:      req.Story,
		Input:      req.Input,
		Output:     req.Output,
		Notes:      req.Notes,
		Scoring:    req.Scoring,
		Talk:       req.Talk,
		Example:    req.Example,
	})
	return
}

func (h *taskHttpHandler) DeleteStatementImage(ctx context.Context,
	req struct{}, params map[string]string,
) (response struct{}, err error) {
	taskId := params["taskId"]
	filename := params["filename"]

	err = h.taskSrvc.DeleteStatementImage(ctx, taskId, filename)
	if err != nil {
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	return
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
	_, imageFilenameExt, vErr := fname.ValidateUploadedImageFilename(uploadedFilename)
	if vErr != nil {
		code := "invalid_filename"
		if ve, ok := vErr.(*fname.FilenameValidationError); ok {
			code = ve.Code
		}
		httpjson.Error(w, vErr.Error(), http.StatusBadRequest, code)
		return
	}

	// get specified and detected MIME types
	_, imageMimeType, err := mimetype.GetUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get MIME types: %v", err)
		errCode := "failed_to_get_mimes"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// can we somehow check that imageFilenameExt matches the mime type?
	if !mimetype.IsExtensionValidForMIME(imageFilenameExt, imageMimeType) {
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
		writeHttpJsonError(w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = httpjson.Success(w, uri)
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}
