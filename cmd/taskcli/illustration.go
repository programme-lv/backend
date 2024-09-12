package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/task"
	"github.com/wailsapp/mimetype"
)

func uploadIllustrationImage(image *fstask.Asset, taskSrvc *task.TaskService) (s3key string, err error) {
	// Compress the image before uploading
	compressedIllustrationImg, err := compressImage(image.Content, 600)
	if err != nil {
		log.Fatalf("Failed to compress image: %v", err)
	}

	mType := mime.TypeByExtension(filepath.Ext(image.RelativePath))
	if mType == "" {
		mTypeDetected := mimetype.Detect(compressedIllustrationImg)
		if mTypeDetected == nil {
			return "", fmt.Errorf("failed to detect file type: %v", err)
		}
		mType = mTypeDetected.String()
	}
	if err != nil {
		return "", fmt.Errorf("failed to detect file type: %v", err)
	}

	s3key, err = taskSrvc.UploadIllustrationImg(mType, compressedIllustrationImg)
	if err != nil {
		return "", fmt.Errorf("failed to upload illustration to S3: %v", err)
	}

	return s3key, nil
}

// func getHexEncodedSHA256Hash(value []byte) string {
// 	h := sha256.New()
// 	h.Write(value)
// 	hash := fmt.Sprintf("%x", h.Sum(nil))
// 	return hash
// }

func compressImage(imgContent []byte, maxWidth uint) ([]byte, error) {
	mime := mimetype.Detect(imgContent)

	var img image.Image
	var err error

	switch mime.String() {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgContent))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgContent))
	// Add more cases if needed for other image formats like GIF, BMP, etc.
	default:
		return nil, fmt.Errorf("unsupported image format: %s", mime.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	width := uint(img.Bounds().Dx())
	if width > maxWidth {
		width = maxWidth
	}
	// Resize the image to the specified width, preserving aspect ratio
	resizedImg := resize.Resize(width, 0, img, resize.Lanczos3)

	var compressedImg bytes.Buffer
	// Encode the resized image to JPEG format with specified quality
	err = jpeg.Encode(&compressedImg, resizedImg, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to JPEG: %v", err)
	}

	return compressedImg.Bytes(), nil
}
