package mimetype

import (
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
)

// GetUploadedFileMIMEs returns (clientMime, detectedMime)
func GetUploadedFileMIMEs(file multipart.File, header *multipart.FileHeader) (string, string, error) {
	clientMime := header.Header.Get("Content-Type")
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return clientMime, "", err
	}
	detectedMime := http.DetectContentType(buf[:n])
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}
	return clientMime, detectedMime, nil
}

// IsExtensionValidForMIME validates extension against mime type
func IsExtensionValidForMIME(ext string, mimeType string) bool {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	mimeToExtensions := map[string][]string{
		"image/jpeg":    {".jpg", ".jpeg"},
		"image/png":     {".png"},
		"image/gif":     {".gif"},
		"image/webp":    {".webp"},
		"image/svg+xml": {".svg"},
		"image/bmp":     {".bmp"},
		"image/tiff":    {".tif", ".tiff"},
	}
	allowed, exists := mimeToExtensions[mimeType]
	if !exists {
		return false
	}
	return slices.Contains(allowed, ext)
}
