package img

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
)

// GetImageDimensions returns width and height for supported mime types
func GetImageDimensions(imageBytes []byte, mimeType string) (int, int, error) {
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
