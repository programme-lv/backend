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
	"path/filepath"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/backend/common/srvcerror"
)

// validateTaskId validates the TaskZip v1 task ID format.
func validateTaskId(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("task ID cannot be empty")
	}
	if len(id) > 64 {
		return fmt.Errorf("task ID too long, max 64 chars")
	}
	for i, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r == '-' && i > 0)) {
			return fmt.Errorf("task ID must start with a lowercase letter or digit and contain only lowercase letters, digits, and hyphens")
		}
	}
	return nil
}

func (ts *taskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) srvcerror.E {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("update statement", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

func (ts *taskSrvc) CreateTask(ctx context.Context, task Task) srvcerror.E {
	for i := range task.MdImages {
		task.MdImages[i].S3Key = taskStatementImageStoredKey(task.MdImages[i].S3Key)
	}
	err := ts.repo.CreateTask(ctx, task)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("failed to create task", "error", err)
		return NewErrorInternalServerError()
	}
	return nil
}

// DeleteTask deletes a task and all its related data.
// Note: This only deletes data from the database. S3 cleanup should be handled separately.
func (ts *taskSrvc) DeleteTask(ctx context.Context, shortId string) srvcerror.E {
	l := ts.logger(ctx)
	err := ts.repo.DeleteTask(ctx, shortId)
	if err != nil {
		l.Error("delete task", "task_id", shortId, "error", err)
		return NewErrorInternalServerError()
	}
	l.Info("task deleted successfully", "task_id", shortId)
	return nil
}

// Object key format: "illustrations/<sha2>.<ext>"
// returns the stored key for the uploaded illustration image.
func (ts *taskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, srvcerror.E) {
	l := ts.logger(ctx)
	sha2 := Sha2Hex(body)
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
	storedKey := fmt.Sprintf("%s%s", sha2, ext)
	_, err = ts.publicStore.Upload(body, taskIllustrationObjectKey(storedKey), mimeType)
	if err != nil {
		l.Error("upload illustration", "error", err)
		return "", NewErrorInternalServerError()
	}
	return storedKey, nil
}

// upload and return an object key. use a new uuid in the filename instead of a sha256 hash.
func (ts *taskSrvc) UploadOgFileArchive(ctx context.Context, zipBytes []byte) (string, srvcerror.E) {
	archiveUuid := uuid.New().String()
	s3Key := fmt.Sprintf("og-file-archives/%s.zip", archiveUuid)
	_, err := ts.publicStore.Upload(zipBytes, s3Key, "application/zip")
	if err != nil {
		ts.logger(ctx).Error("upload og file archive", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Key, nil
}

func (ts *taskSrvc) DownloadOgFileArchive(ctx context.Context, s3Key string) ([]byte, srvcerror.E) {
	body, err := ts.publicStore.Download(s3Key)
	if err != nil {
		ts.logger(ctx).Error("download og file archive", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return body, nil
}

// Object key format: "md-images/<taskId>/<sha256-prefix>.<extension>"
func (ts *taskSrvc) UploadStatementImage(ctx context.Context, taskId string, imgFilename string, imageMimeType string, body []byte) (string, srvcerror.E) {
	l := ts.logger(ctx)

	// get the file extension from the mime type, e.g. "image/png" -> ".png"
	ext, err := MimeToFileExt(imageMimeType)
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

	storedKey := fmt.Sprintf("%s/%s%s", taskId, Sha2Hex(body)[:12], ext)
	s3Uri, err := ts.publicStore.Upload(body, taskStatementImageObjectKey(storedKey), imageMimeType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return "", NewErrorInternalServerError()
	}

	// update the task with the new image
	err = ts.repo.AddStatementImg(ctx, taskId, StatementImage{
		S3Key:     storedKey,
		Filename:  imgFilename,
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: len(body),
	})
	if err != nil {
		l.Error("failed to add statement image to db", "error", err)
		return "", NewErrorInternalServerError()
	}
	return s3Uri, nil
}

func MimeToFileExt(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extension not found")
	}
	return exts[0], nil
}

func MimeFromFname(fname string) (string, error) {
	ext := filepath.Ext(fname)
	if ext == "" {
		return "", fmt.Errorf("file extension not found")
	}
	return mime.TypeByExtension(ext), nil
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

func GetTestfileS3Key(body []byte) string {
	shaHex := Sha2Hex(body)
	return fmt.Sprintf("%s.zst", shaHex)
}

// UploadTestFile uploads a test input or output to S3 after compressing it with Zstandard.
// If The test already exists, it returns no error and does nothing.
//
// The S3 key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *taskSrvc) UploadTestFile(ctx context.Context, body []byte) srvcerror.E {
	l := ts.logger(ctx)
	s3Key := GetTestfileS3Key(body)
	mediaType := "application/zstd"

	exists, err := ts.testfileStore.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if object exists in S3", "error", err)
		return NewErrorInternalServerError()
	}

	if exists {
		return nil
	}

	zstdCompressed, err := CompressWithZstd(body)
	if err != nil {
		l.Error("failed to compress data", "error", err)
		return NewErrorInternalServerError()
	}

	_, err = ts.testfileStore.Upload(zstdCompressed, s3Key, mediaType)
	if err != nil {
		l.Error("failed to upload to S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// CompressWithZstd compresses the given data using Zstandard compression.
// It returns the compressed data or an error if the compression fails.
func CompressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

func MustCompressWithZstd(data []byte) []byte {
	compressed, err := CompressWithZstd(data)
	if err != nil {
		panic(err)
	}
	return compressed
}

// DecompressWithZstd decompresses data that was compressed with Zstandard.
// It returns the decompressed data or an error if the decompression fails.
func DecompressWithZstd(compressedData []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}
	return decompressed, nil
}

func Sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}

// DeleteStatementImage implements TaskSrvcClient.
// It deletes an image from both S3 and the database.
func (ts *taskSrvc) DeleteStatementImage(ctx context.Context, taskId string, filename string) srvcerror.E {
	l := ts.logger(ctx)

	// First, get the task to find the image with the specified filename
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
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
		return NewErrorImageNotFound(filename)
	}

	s3Key := taskStatementImageObjectKey(targetImage.S3Key)

	// Check if the image exists in S3
	exists, err := ts.publicStore.Exists(s3Key)
	if err != nil {
		l.Error("check if image exists in S3", "error", err)
		return NewErrorInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return NewErrorInternalServerError()
	}

	// Delete the image from the database first
	err = ts.repo.DeleteStatementImg(ctx, taskId, filename)
	if err != nil {
		l.Error("delete image from database", "error", err)
		return NewErrorInternalServerError()
	}

	// Delete the image from S3
	err = ts.publicStore.Delete(s3Key)
	if err != nil {
		l.Error("delete image from S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// DeleteIllustrationImg implements TaskSrvcClient.
// It deletes an illustration image from both S3 and the database.
func (ts *taskSrvc) DeleteIllustrationImg(ctx context.Context, taskId string) srvcerror.E {
	l := ts.logger(ctx)

	// First, get the task to find the illustration image S3 key
	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
		return NewErrorFailedToGetTaskFromDb(taskId)
	}

	// Check if there's an illustration image to delete
	if t.IllustrImg.S3Key == "" {
		l.Error("no illustration image found for task", "task_id", taskId)
		return NewErrorImageNotFound(taskId)
	}

	s3Key := taskIllustrationObjectKey(t.IllustrImg.S3Key)

	// Check if the image exists in S3
	exists, err := ts.publicStore.Exists(s3Key)
	if err != nil {
		l.Error("failed to check if image exists in S3", "error", err)
		return NewErrorInternalServerError()
	}
	if !exists {
		l.Error("image does not exist in S3", "s3_key", s3Key)
		return NewErrorInternalServerError()
	}

	// Update the database to remove illustration image fields
	emptyImg := IllustrationImage{
		S3Key:     "",
		WidthPx:   0,
		HeightPx:  0,
		SzInBytes: 0,
	}
	err = ts.repo.UpdateIllustrationImg(ctx, taskId, emptyImg)
	if err != nil {
		l.Error("failed to update illustration image in database", "error", err)
		return NewErrorInternalServerError()
	}

	// Delete the image from S3
	err = ts.publicStore.Delete(s3Key)
	if err != nil {
		l.Error("failed to delete image from S3", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

// UpdateIllustrationImg implements TaskSrvcClient.
// It updates the illustration image information in the database.
func (ts *taskSrvc) UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) srvcerror.E {
	l := ts.logger(ctx)
	img.S3Key = taskIllustrationStoredKey(img.S3Key)

	err := ts.repo.UpdateIllustrationImg(ctx, taskId, img)
	if err != nil {
		l.Error("failed to update illustration image in database", "error", err)
		return NewErrorInternalServerError()
	}

	return nil
}

func (ts *taskSrvc) DownloadTestFile(ctx context.Context, testFileSha256 string) ([]byte, srvcerror.E) {
	logger := ts.logger(ctx)

	// Try cache first
	if content, found := ts.testCache.getFromCache(testFileSha256); found {
		logger.Debug("test file cache hit (direct)", "sha256", testFileSha256)
		return content, nil
	}

	// Use singleflight to prevent thundering herd for the same key
	v, err, _ := ts.dlGroup.Do(testFileSha256, func() (any, error) {
		// Re-check cache inside the singleflight window
		if content, found := ts.testCache.getFromCache(testFileSha256); found {
			return content, nil
		}

		// Download compressed from S3 by key <sha>.zst
		s3Key := fmt.Sprintf("%s.zst", testFileSha256)
		compressed, err := ts.testfileStore.Download(s3Key)
		if err != nil {
			logger.Error("failed to download test file from S3", "sha256", testFileSha256, "s3_key", s3Key, "error", err)
			return nil, NewErrorInternalServerError()
		}

		// Decompress
		content, err := DecompressWithZstd(compressed)
		if err != nil {
			logger.Error("failed to decompress test file", "sha256", testFileSha256, "error", err)
			return nil, NewErrorInternalServerError()
		}

		// Store in cache
		ts.testCache.storeInCache(testFileSha256, content, logger)
		return content, nil
	})
	if err != nil {
		if se, ok := err.(srvcerror.E); ok {
			return nil, se
		}
		logger.Error("unexpected error type from singleflight", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return v.([]byte), nil
}
