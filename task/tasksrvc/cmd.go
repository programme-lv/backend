package tasksrvc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"mime"

	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/backend/task/taskdomain"
)

func (ts *TaskSrvc) PutTask(ctx context.Context, task *taskdomain.Task) error {
	key := fmt.Sprintf("%s.json", task.ShortId)

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	_, err = ts.s3TaskBucket.Upload(data, key, "application/json")
	if err != nil {
		return fmt.Errorf("failed to upload task: %w", err)
	}

	return nil
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

// S3 key format: "task-md-images/<sha2>.<extension>"
func (ts *TaskSrvc) UploadMarkdownImage(ctx context.Context, mimeType string, body []byte) (url string, err error) {
	sha2 := ts.Sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extennsion not found")
	}
	ext := exts[0]
	s3Key := fmt.Sprintf("%s/%s%s", "task-md-images", sha2, ext)
	return ts.s3PublicBucket.Upload(body, s3Key, mimeType)
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
