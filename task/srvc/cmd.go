package srvc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

func (ts *TaskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) error {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to update statement", "error", err)
		return ErrInternalServerError()
	}

	return nil
}

func (ts *TaskSrvc) CreateTask(ctx context.Context, task Task) error {
	err := ts.repo.CreateTask(ctx, task)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to create task", "error", err)
		return ErrInternalServerError()
	}
	return nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-pdf-statements/<sha2>.pdf"
func (ts *TaskSrvc) UploadStatementPdf(ctx context.Context, body []byte) (string, error) {
	l := ts.logger(ctx)
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	url, err := ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
	if err != nil {
		l.Error("failed to upload PDF to S3", "error", err)
		return "", ErrInternalServerError()
	}
	return url, nil
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-illustrations/<sha2>.<ext>"
func (ts *TaskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (url string, err error) {
	l := ts.logger(ctx)
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		l.Error("failed to get file extension", "error", err)
		return "", NewErrorImageFileExtFromMimeType(mimeType)
	}
	if len(exts) == 0 {
		l.Error("file extension not found", "mime_type", mimeType)
		return "", NewErrorImageFileExtFromMimeType(mimeType)
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	url, err = ts.s3PublicBucket.Upload(body, s3Key, mimeType)
	if err != nil {
		l.Error("failed to upload illustration to S3", "error", err)
		return "", ErrInternalServerError()
	}
	return url, nil
}

// S3 key format: "task-md-images/<uuid>.<extension>"
// returns s3 uri, e.g. s3://proglv-public/task/<taskId>/md-images/<uuid>.png
func (ts *TaskSrvc) UploadStatementImage(ctx context.Context, taskId string, imgFilename string, imageMimeType string, body []byte) (url string, err error) {
	l := ts.logger(ctx)

	// get the file extension from the mime type, e.g. "image/png" -> ".png"
	ext, err := getImgExt(imageMimeType)
	if err != nil {
		l.Error("failed to get image file extension from mime type", "error", err)
		return "", NewErrorImageFileExtFromMimeType(imageMimeType)
	}

	// get the image width and height in pixels
	width, height, err := getImgWidthHeighPx(body, imageMimeType)
	if err != nil {
		l.Error("failed to get image width and height", "error", err)
		return "", NewErrorGetImageWidthAndHeight()
	}

	// verify that the image heas reasonable dimensions
	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		return "", NewErrorImageInadequateDimensions()
	}

	// find the task to verify that it exists and the an image with the same filename does not exist
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("failed to get corresponding task", "error", err)
		return "", NewErrorFailedToGetTaskFromDb(taskId)
	}
	for _, img := range t.MdImages {
		if img.Filename == imgFilename {
			return "", NewErrorImageAlreadyExists(imgFilename)
		}
	}

	// generate a new UUID for the image (to avoid collision and reduce complexity when renaming semantic filenames), and upload it to S3
	newImgUuid := uuid.New().String()
	s3Key := fmt.Sprintf("task/%s/md-images/%s%s", taskId, newImgUuid, ext)
	s3Uri, err := ts.s3PublicBucket.Upload(body, s3Key, imageMimeType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return "", ErrInternalServerError()
	}

	// update the task with the new image
	err = ts.repo.AddStatementImg(ctx, taskId, StatementImage{
		S3Key:     s3Key,
		Filename:  imgFilename,
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: len(body),
	})
	if err != nil {
		l.Error("failed to add statement image to db", "error", err)
		return "", ErrInternalServerError()
	}
	return s3Uri, nil
}

func getImgExt(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extension not found")
	}
	return exts[0], nil
}

func getImgWidthHeighPx(body []byte, mimeType string) (int, int, error) {
	reader := bytes.NewReader(body)

	switch mimeType {
	case "image/png":
		img, err := png.Decode(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode PNG image: %w", err)
		}
		return img.Bounds().Dx(), img.Bounds().Dy(), nil
	case "image/jpeg", "image/jpg":
		// For JPEGs, use the more efficient DecodeConfig which just reads the header
		config, err := jpeg.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode JPEG image: %w", err)
		}
		return config.Width, config.Height, nil
	default:
		// Fallback for other image types
		config, _, err := image.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode image: %w", err)
		}
		return config.Width, config.Height, nil
	}
}

// UploadTestFile uploads a test input or output to S3 after compressing it with Zstandard.
// If The test already exists, it returns no error and does nothing.
//
// The S3 key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *TaskSrvc) UploadTestFile(ctx context.Context, body []byte) error {
	l := ts.logger(ctx)
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s.zst", shaHex)
	mediaType := "application/zstd"

	exists, err := ts.s3TestfileBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if object exists in S3", "error", err)
		return ErrInternalServerError()
	}

	if exists {
		return nil
	}

	zstdCompressed, err := compressWithZstd(body)
	if err != nil {
		l.Error("failed to compress data", "error", err)
		return ErrInternalServerError()
	}

	_, err = ts.s3TestfileBucket.Upload(zstdCompressed, s3Key, mediaType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return ErrInternalServerError()
	}

	return nil
}

// compressWithZstd compresses the given data using Zstandard compression.
// It returns the compressed data or an error if the compression fails.
func compressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

func (ts *TaskSrvc) Sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

// DeleteStatementImage implements TaskSrvcClient.
// It deletes an image from both S3 and the database.
func (ts *TaskSrvc) DeleteStatementImage(ctx context.Context, taskId string, filename string) error {
	l := ts.logger(ctx)

	// First, get the task to find the image with the specified filename
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("failed to get task", "error", err)
		return NewErrorFailedToGetTaskFromDb(taskId)
	}

	// Find the image with the specified filename
	var targetImage *StatementImage
	for _, img := range t.MdImages {
		if img.Filename == filename {
			targetImage = &img
			break
		}
	}

	if targetImage == nil {
		l.Error("image with filename not found", "filename", filename)
		return fmt.Errorf("image with filename %s does not exist for task %s", filename, taskId)
	}

	// The S3 key is already stored directly in the database
	s3Key := targetImage.S3Key

	// Check if the image exists in S3
	exists, err := ts.s3PublicBucket.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if image exists in S3", "error", err)
		return ErrInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return ErrInternalServerError()
	}

	// Delete the image from the database first
	err = ts.repo.DeleteStatementImg(ctx, taskId, filename)
	if err != nil {
		l.Error("failed to delete image from database", "error", err)
		return ErrInternalServerError()
	}

	// Delete the image from S3
	err = ts.s3PublicBucket.Delete(s3Key)
	if err != nil {
		l.Error("failed to delete image from S3", "error", err)
		return ErrInternalServerError()
	}

	return nil
}
