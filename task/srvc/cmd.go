package srvc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"mime"
	"strings"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

func (ts *TaskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) error {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		return fmt.Errorf("failed to update statement: %w", err)
	}

	return nil
}

func (ts *TaskSrvc) CreateTask(ctx context.Context, task Task) error {
	return ts.repo.CreateTask(ctx, task)
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-pdf-statements/<sha2>.pdf"
func (ts *TaskSrvc) UploadStatementPdf(ctx context.Context, body []byte) (string, error) {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s/%s.pdf", "task-pdf-statements", shaHex)
	return ts.s3PublicBucket.Upload(body, s3Key, "application/pdf")
}

// S3 bucket: "proglv-public" (as of 2024-09-29)
// S3 key format: "task-illustrations/<sha2>.<ext>"
func (ts *TaskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (url string, err error) {
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extennsion not found")
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-illustrations", sha2, ext)
	return ts.s3PublicBucket.Upload(body, s3Key, mimeType)
}

// S3 key format: "task-md-images/<uuid>.<extension>"
// returns s3 uri, e.g. s3://proglv-public/task/<taskId>/md-images/<uuid>.png
func (ts *TaskSrvc) UploadStatementImage(ctx context.Context, taskId string, semanticFilename string, imageMimeType string, body []byte) (url string, err error) {
	// get the file extension from the mime type, e.g. "image/png" -> ".png"
	ext, err := getImgExt(imageMimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}

	// get the image width and height in pixels
	width, height, err := getImgWidthHeighPx(body, imageMimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get image width and height: %w", err)
	}

	// verify that the image heas reasonable dimensions
	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		return "", fmt.Errorf("image is too large or has no dimensions")
	}

	// find the task just to verify that it exists
	_, err = ts.repo.GetTask(ctx, taskId)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

	// generate a new UUID for the image (to avoid collision and reduce complexity when renaming semantic filenames), and upload it to S3
	newImgUuid := uuid.New().String()
	s3Key := fmt.Sprintf("task/%s/md-images/%s%s", taskId, newImgUuid, ext)
	s3Uri, err := ts.s3PublicBucket.Upload(body, s3Key, imageMimeType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// update the task with the new image
	err = ts.repo.AddStatementImg(ctx, taskId, StatementImage{
		S3Uri:    s3Uri,
		Filename: semanticFilename,
		WidthPx:  width,
		HeightPx: height,
	})
	if err != nil {
		return "", fmt.Errorf("failed to add statement imgage: %w", err)
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
	if mimeType == "image/png" {
		img, err := png.Decode(bytes.NewReader(body))
		if err != nil {
			return 0, 0, fmt.Errorf("failed to decode image: %w", err)
		}
		return img.Bounds().Dx(), img.Bounds().Dy(), nil
	}
	img, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}
	return img.Width, img.Height, nil
}

// UploadTestFile uploads a test input or output to S3 after compressing it with Zstandard.
// If The test already exists, it returns no error and does nothing.
//
// The S3 key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *TaskSrvc) UploadTestFile(ctx context.Context, body []byte) error {
	shaHex := ts.Sha2Hex(body)
	s3Key := fmt.Sprintf("%s.zst", shaHex)
	mediaType := "application/zstd"

	exists, err := ts.s3TestfileBucket.Exists(s3Key)
	if err != nil {
		return fmt.Errorf("failed to check if object exists in S3: %w", err)
	}

	if exists {
		return nil
	}

	zstdCompressed, err := compressWithZstd(body)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	_, err = ts.s3TestfileBucket.Upload(zstdCompressed, s3Key, mediaType)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
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
func (ts *TaskSrvc) DeleteStatementImage(ctx context.Context, taskId string, s3Uri string) error {
	// Extract the S3 key from the URI
	// s3Uri format: s3://proglv-public/task/<taskId>/md-images/<uuid>.png
	s3Key := strings.TrimPrefix(s3Uri, "s3://"+ts.s3PublicBucket.Bucket()+"/")

	// Check if the image exists in S3
	exists, err := ts.s3PublicBucket.Exists(s3Key)
	if err != nil {
		return fmt.Errorf("failed to check if image exists in S3: %w", err)
	}
	if !exists {
		return fmt.Errorf("image with key %s does not exist in S3", s3Key)
	}

	// Delete the image from the database first
	err = ts.repo.DeleteStatementImg(ctx, taskId, s3Uri)
	if err != nil {
		return fmt.Errorf("failed to delete image from database: %w", err)
	}

	// Delete the image from S3
	err = ts.s3PublicBucket.Delete(s3Key)
	if err != nil {
		return fmt.Errorf("failed to delete image from S3: %w", err)
	}

	return nil
}
