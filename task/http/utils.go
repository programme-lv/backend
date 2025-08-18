package http

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
)

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
