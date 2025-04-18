package http

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/user/auth"
)

func (h *TaskHttpHandler) DeleteStatementImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.WriteErrorJson(w, "Can't delete statement image as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}
}

func (h *TaskHttpHandler) UploadStatementImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok || claims == nil || claims.Username != "admin" {
		httpjson.WriteErrorJson(w, "Can't edit statement as non-admin user", http.StatusUnauthorized, "unauthorized")
		return
	}
	taskId := chi.URLParam(r, "taskId")

	err := r.ParseMultipartForm(10 << 20) // max 10MB
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse multipart form (maybe the image is too large?): %v", err)
		errCode := "failed_to_parse_multipart_form"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// get image as byte array
	image, header, err := r.FormFile("image")
	if err != nil {
		errMsg := fmt.Sprintf("failed to get image: %v", err)
		errCode := "failed_to_get_image"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
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
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	if len(filenameWithoutExt) > 100 {
		errMsg := fmt.Sprintf("filename is too long (max 100 characters): %s", filenameWithoutExt)
		errCode := "filename_too_long"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}
	cantContain := []string{"CON", "PRN", "AUX", "NUL", "COM", "LPT"}
	if len(filenameWithoutExt) < 4 && slices.Contains(cantContain, filenameWithoutExt) {
		errMsg := fmt.Sprintf("invalid filename (may contain reserved filenames): %s", filenameWithoutExt)
		errCode := "invalid_filename"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// get specified and detected MIME types
	_, imageMimeType, err := getUploadedFileMIMEs(image, header)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get MIME types: %v", err)
		errCode := "failed_to_get_mimes"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	// can we somehow check that imageFilenameExt matches the mime type?
	if !isExtensionValidForMIME(imageFilenameExt, imageMimeType) {
		errMsg := fmt.Sprintf("file extension '%s' does not match detected MIME type '%s'", imageFilenameExt, imageMimeType)
		errCode := "invalid_file_extension"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read image: %v", err)
		errCode := "failed_to_read_image"
		httpjson.WriteErrorJson(w, errMsg, http.StatusBadRequest, errCode)
		return
	}

	uri, err := h.taskSrvc.UploadStatementImage(r.Context(), taskId, uploadedFilename, imageMimeType, imageBytes)
	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	err = httpjson.WriteSuccessJson(w, uri)
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
