package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/wailsapp/mimetype"
)

// UploadIllustrationImage compresses the image and uploads it to S3.
// It returns the S3 key or an error if the process fails.
func UploadIllustrationImage(asset *fstask.Asset, taskService *tasksrvc.TaskService) (string, error) {
	compressedImage, err := compressImage(asset.Content, 600)
	if err != nil {
		return "", fmt.Errorf("failed to compress image: %w", err)
	}

	mType := mime.TypeByExtension(filepath.Ext(asset.RelativePath))
	if mType == "" {
		detectedType := mimetype.Detect(compressedImage)
		if detectedType == nil {
			return "", fmt.Errorf("failed to detect file type")
		}
		mType = detectedType.String()
	}

	s3Key, err := taskService.UploadIllustrationImg(mType, compressedImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload illustration to S3: %w", err)
	}

	return s3Key, nil
}

// compressImage resizes and compresses the image to the specified maximum width.
// It returns the compressed image bytes or an error if the process fails.
func compressImage(imgContent []byte, maxWidth uint) ([]byte, error) {
	mType := mimetype.Detect(imgContent)
	if mType == nil {
		return nil, fmt.Errorf("unknown image type")
	}

	var img image.Image
	var err error

	switch mType.String() {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgContent))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgContent))
	default:
		return nil, fmt.Errorf("unsupported image format: %s", mType.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize the image while maintaining aspect ratio
	width := uint(img.Bounds().Dx())
	if width > maxWidth {
		width = maxWidth
	}
	resizedImg := resize.Resize(width, 0, img, resize.Lanczos3)

	var compressedImg bytes.Buffer
	// Encode the resized image to JPEG format with quality 85
	err = jpeg.Encode(&compressedImg, resizedImg, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	return compressedImg.Bytes(), nil
}
