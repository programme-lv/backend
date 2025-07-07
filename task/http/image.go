package http

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/httpjson"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user/auth"
)

func (h *TaskHttpHandler) DeleteStatementImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.Error(w, "Can't delete statement image as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}

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

func (h *TaskHttpHandler) UploadStatementImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.Error(w, "Can't edit statement as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}
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

// getUploadedFileMIMEs reads up to 512 bytes from the provided multipart.File
// to sniff the actual MIME type, and also returns the client-reported one.
// It resets the file's read pointer before returning.
//
//	file:   the opened multipart.File from r.FormFile
//	header: the accompanying *multipart.FileHeader
//
// Returns (clientMime, detectedMime, error).
func getUploadedFileMIMEs(file multipart.File, header *multipart.FileHeader) (string, string, error) {
	// 1) client‐reported
	clientMime := header.Header.Get("Content-Type")

	// 2) server‐sniffed
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return clientMime, "", err
	}
	detectedMime := http.DetectContentType(buf[:n])

	// reset reader so caller can re-read the file if needed
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	return clientMime, detectedMime, nil
}

// isExtensionValidForMIME checks if the file extension matches the MIME type
func isExtensionValidForMIME(ext string, mimeType string) bool {
	// Convert extension to lowercase and ensure it starts with a dot
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Map of MIME types to allowed extensions
	mimeToExtensions := map[string][]string{
		"image/jpeg":    {".jpg", ".jpeg"},
		"image/png":     {".png"},
		"image/gif":     {".gif"},
		"image/webp":    {".webp"},
		"image/svg+xml": {".svg"},
		"image/bmp":     {".bmp"},
		"image/tiff":    {".tif", ".tiff"},
	}

	// Check if the MIME type exists in our map
	allowedExtensions, exists := mimeToExtensions[mimeType]
	if !exists {
		// If we don't have this MIME type in our map, we can't validate it
		return false
	}

	// Check if the extension is in the list of allowed extensions for this MIME type
	return slices.Contains(allowedExtensions, ext)
}

func (h *TaskHttpHandler) UploadIllustrationImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.Error(w, "Can't upload illustration image as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}

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

	// get specified and detected MIME types
	_, imageMimeType, err := getUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get MIME types: %v", err)
		errCode := "failed_to_get_mimes"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// validate that this is an image MIME type
	if !strings.HasPrefix(imageMimeType, "image/") {
		errMsg := fmt.Sprintf("uploaded file is not an image (detected MIME type: %s)", imageMimeType)
		errCode := "invalid_mime_type"
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

	// Upload the illustration image to S3
	uri, err := h.taskSrvc.UploadIllustrationImg(r.Context(), imageMimeType, imageBytes)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	// Get image dimensions
	width, height, err := getImageDimensions(imageBytes, imageMimeType)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get image dimensions: %v", err)
		errCode := "failed_to_get_image_dimensions"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// Verify reasonable dimensions
	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		errMsg := "image dimensions are inadequate (must be between 1x1 and 2000x2000 pixels)"
		errCode := "inadequate_image_dimensions"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// Create the S3 key using the same logic as the service
	sha2 := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	exts, err := mime.ExtensionsByType(imageMimeType)
	if err != nil || len(exts) == 0 {
		errMsg := fmt.Sprintf("failed to get file extension for MIME type: %s", imageMimeType)
		errCode := "invalid_mime_type"
		httpjson.Error(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("task-illustrations/%s%s", sha2, ext)

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
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	h.cache.Delete(taskGetCacheKey(taskId))

	err = httpjson.Success(w, map[string]string{"url": uri})
	if err != nil {
		slog.Error("failed to write success json", "error", err)
	}
}

func (h *TaskHttpHandler) DeleteIllustrationImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.Error(w, "Can't delete illustration image as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}

	taskId := chi.URLParam(r, "taskId")

	err := h.taskSrvc.DeleteIllustrationImg(r.Context(), taskId)
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

func getImageDimensions(imageBytes []byte, mimeType string) (int, int, error) {
	var width, height int

	switch mimeType {
	case "image/jpeg":
		img, err := jpeg.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return 0, 0, err
		}
		width, height = img.Bounds().Dx(), img.Bounds().Dy()
	case "image/png":
		img, err := png.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return 0, 0, err
		}
		width, height = img.Bounds().Dx(), img.Bounds().Dy()
	default:
		return 0, 0, fmt.Errorf("unsupported image format: %s", mimeType)
	}

	return width, height, nil
}
