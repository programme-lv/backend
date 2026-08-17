package srvc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/srvcerror"
)

const testfileDownloadURLValidity = time.Hour

func testfileObjectKey(body []byte) string {
	shaHex := sha2Hex(body)
	return fmt.Sprintf("%s.zst", shaHex)
}

// UploadTestFile stores a test input or output after compressing it with Zstandard.
// If the object already exists, it returns no error and does nothing.
// The object key is the SHA256 hash of the uncompressed body with a .zst extension.
func (ts *taskSrvc) UploadTestFile(ctx context.Context, body []byte) srvcerror.E {
	l := ts.logger(ctx)
	objectKey := testfileObjectKey(body)
	mediaType := "application/zstd"

	exists, err := ts.testfileStore.Exists(objectKey)
	if err != nil {
		l.Error("check if test file exists", "error", err)
		return srvcerror.InternalServerError()
	}

	if exists {
		return nil
	}

	zstdCompressed, err := compressWithZstd(body)
	if err != nil {
		l.Error("compress data", "error", err)
		return srvcerror.InternalServerError()
	}

	_, err = ts.testfileStore.Upload(zstdCompressed, objectKey, mediaType)
	if err != nil {
		l.Error("upload test file", "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}

// DownloadTestFile returns the uncompressed test file for sha256.
// Concurrent downloads of the same key are coalesced with singleflight.
func (ts *taskSrvc) DownloadTestFile(ctx context.Context, testFileSha256 string) ([]byte, srvcerror.E) {
	logger := ts.logger(ctx)

	if content, found := ts.testCache.getFromCache(testFileSha256); found {
		logger.Debug("test file cache hit (direct)", "sha256", testFileSha256)
		return content, nil
	}

	v, err, _ := ts.dlGroup.Do(testFileSha256, func() (any, error) {
		if content, found := ts.testCache.getFromCache(testFileSha256); found {
			return content, nil
		}

		objectKey := fmt.Sprintf("%s.zst", testFileSha256)
		compressed, err := ts.testfileStore.Download(objectKey)
		if err != nil {
			logger.Error("download test file", "sha256", testFileSha256, "object_key", objectKey, "error", err)
			return nil, srvcerror.InternalServerError()
		}

		content, err := decompressWithZstd(compressed)
		if err != nil {
			logger.Error("decompress test file", "sha256", testFileSha256, "error", err)
			return nil, srvcerror.InternalServerError()
		}

		ts.testCache.storeInCache(testFileSha256, content, logger)
		return content, nil
	})
	if err != nil {
		if se, ok := err.(srvcerror.E); ok {
			return nil, se
		}
		logger.Error("unexpected error type from singleflight", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	return v.([]byte), nil
}

// GetTestDownlUrl returns a time-limited signed URL for a test file.
func (ts *taskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, srvcerror.E) {
	return filestore.SignedTestfileURL(
		ts.apiPublicBaseURL,
		testFileSha256,
		ts.testfileDownloadSigningKey,
		time.Now().Add(testfileDownloadURLValidity),
	), nil
}

func compressWithZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("create zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

func decompressWithZstd(compressedData []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decompress data: %w", err)
	}
	return decompressed, nil
}

func sha2Hex(body []byte) (sha2 string) {
	hash := sha256.Sum256(body)
	sha2 = fmt.Sprintf("%x", hash[:])
	return
}
