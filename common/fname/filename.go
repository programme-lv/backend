package fname

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// FilenameValidationError provides a stable error code for HTTP mapping.
type FilenameValidationError struct {
	Code string
	Msg  string
}

func (e *FilenameValidationError) Error() string { return e.Msg }

// ValidateUploadedImageFilename validates an uploaded filename's base part and returns the base and extension.
// Rules:
// - base must match ^[a-zA-Z0-9_-]+$
// - base length <= 100
// - if base length < 4, it must not be in the reserved list (CON, PRN, AUX, NUL, COM, LPT)
func ValidateUploadedImageFilename(uploaded string) (base string, ext string, err error) {
	base = strings.TrimSuffix(uploaded, filepath.Ext(uploaded))
	ext = filepath.Ext(uploaded)

	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !allowedChars.MatchString(base) {
		return "", "", &FilenameValidationError{
			Code: "invalid_filename",
			Msg:  fmt.Sprintf("invalid filename (only alphanumeric characters, underscores, and hyphens are allowed): %s", uploaded),
		}
	}
	if len(base) > 100 {
		return "", "", &FilenameValidationError{
			Code: "filename_too_long",
			Msg:  fmt.Sprintf("filename is too long (max 100 characters): %s", base),
		}
	}
	cantContain := []string{"CON", "PRN", "AUX", "NUL", "COM", "LPT"}
	if len(base) < 4 && slices.Contains(cantContain, base) {
		return "", "", &FilenameValidationError{
			Code: "invalid_filename",
			Msg:  fmt.Sprintf("invalid filename (may contain reserved filenames): %s", base),
		}
	}
	return base, ext, nil
}
