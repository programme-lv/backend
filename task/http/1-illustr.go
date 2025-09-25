package http

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/img"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/common/mimetype"
	"github.com/programme-lv/backend/task/srvc"
)

func (h *taskHttpHandler) UploadIllustrationImage(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	err := r.ParseMultipartForm(10 << 20) // max 10MB
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse multipart form (maybe the image is too large?): %v", err)
		errCode := "failed_to_parse_multipart_form"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// get image as byte array
	image, header, err := r.FormFile("image")
	if err != nil {
		errMsg := fmt.Sprintf("failed to get image: %v", err)
		errCode := "failed_to_get_image"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	defer image.Close()

	// get specified and detected MIME types
	_, imageMimeType, err := mimetype.GetUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get MIME types: %v", err)
		errCode := "failed_to_get_mimes"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// validate that this is an image MIME type
	if !strings.HasPrefix(imageMimeType, "image/") {
		errMsg := fmt.Sprintf("uploaded file is not an image (detected MIME type: %s)", imageMimeType)
		errCode := "invalid_mime_type"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read image: %v", err)
		errCode := "failed_to_read_image"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// Get image dimensions
	width, height, err := img.GetImageDimensions(imageBytes, imageMimeType)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get image dimensions: %v", err)
		errCode := "failed_to_get_image_dimensions"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// Verify reasonable dimensions
	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		errMsg := "image dimensions are inadequate (must be between 1x1 and 2000x2000 pixels)"
		errCode := "inadequate_image_dimensions"
		jsonresp.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// Upload the illustration image to S3
	s3Key, err := h.taskSrvc.UploadIllustrationImg(r.Context(), imageMimeType, imageBytes)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	// Update the task with the illustration image information
	sizeInBytes := len(imageBytes)
	illustrationImg := srvc.IllustrationImage{
		S3Key:     s3Key,
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: sizeInBytes,
	}

	err = h.taskSrvc.UpdateIllustrationImg(r.Context(), taskId, illustrationImg)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = jsonresp.Success(w, map[string]string{"s3_key": s3Key})
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}

func (h *taskHttpHandler) DeleteIllustrationImage(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")

	err := h.taskSrvc.DeleteIllustrationImg(r.Context(), taskId)
	if err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = jsonresp.Success(w, map[string]string{"status": "ok"})
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}
