package s3bucket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FileData holds the key and content of an S3 object.
type FileData struct {
	Key     string
	Content []byte
}

type S3Bucket struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	region    string
}

func (bucket *S3Bucket) Region() string {
	return bucket.region
}

func (bucket *S3Bucket) Bucket() string {
	return bucket.bucket
}

func NewS3Bucket(region string, bucket string) (*S3Bucket, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(client)

	return &S3Bucket{
		client:    client,
		presigner: presigner,
		bucket:    bucket,
		region:    region,
	}, nil
}

// Upload uploads the given content to the S3 bucket with the specified key and media type.
// It returns the URI of the uploaded object or an error if the upload fails.
//
// Parameters:
//   - content: The byte slice containing the content to be uploaded.
//   - key: The key (path) under which the content will be stored in the S3 bucket.
//   - mediaType: The MIME type of the content being uploaded.
//
// Returns:
//   - string: The S3 URI of the uploaded object, e.g. s3://proglv-public/task-md-images/<something>.png
//   - error: An error if the upload fails, otherwise nil.
func (bucket *S3Bucket) Upload(content []byte, key string, mediaType string) (string, error) {
	_, err := bucket.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &bucket.bucket,
		Key:         &key,
		Body:        bytes.NewReader(content),
		ContentType: &mediaType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	// Construct the Object URI
	objectURI := fmt.Sprintf("s3://%s/%s", bucket.bucket, key)

	return objectURI, nil
}

func (bucket *S3Bucket) Exists(key string) (bool, error) {
	_, err := bucket.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &bucket.bucket,
		Key:    &key,
	})
	if err != nil {
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == 404 {
			log.Printf("Key: %s does not exist in S3 bucket: %s", key, bucket.bucket)
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return true, nil
}

func (bucket *S3Bucket) Download(key string) ([]byte, error) {
	output, err := bucket.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}
	defer output.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}
	return buf.Bytes(), nil
}

// TODO presign get request
func (bucket *S3Bucket) PresignedURL(key string, expires time.Duration) (string, error) {
	req, err := bucket.presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket.bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}
	return req.URL, nil
}

// ListFiles lists the files in the S3 bucket.
// It returns a slice of file keys or an error if the operation fails.
//
// Parameters:
//   - prefix: An optional prefix to filter the listed objects (e.g., "images/").
//
// Returns:
//   - []string: A slice containing the keys of the objects in the bucket.
//   - error: An error if the listing fails, otherwise nil.
func (bucket *S3Bucket) ListFiles(prefix string) ([]string, error) {
	var keys []string
	input := &s3.ListObjectsV2Input{
		Bucket: &bucket.bucket,
	}

	if prefix != "" {
		input.Prefix = &prefix
	}

	paginator := s3.NewListObjectsV2Paginator(bucket.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// ListAndGetAllFiles lists all files in the S3 bucket and retrieves their contents.
// It returns a slice of FileData containing each file's key and content, or an error if the operation fails.
//
// Parameters:
//   - prefix: An optional prefix to filter the listed objects (e.g., "images/").
//
// Returns:
//   - []FileData: A slice containing the keys and contents of the objects in the bucket.
//   - error: An error if the operation fails, otherwise nil.
func (bucket *S3Bucket) ListAndGetAllFiles(prefix string) ([]FileData, error) {
	// Step 1: List all file keys
	keys, err := bucket.ListFiles(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Step 2: Initialize a slice to hold the file data
	var files []FileData

	// Optional: Use concurrency to download files in parallel for efficiency
	// Here, we'll use a buffered channel to limit the number of concurrent downloads
	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)
	results := make(chan FileData, len(keys))
	errs := make(chan error, len(keys))
	var wg sync.WaitGroup

	// Step 3: Start downloading each file concurrently
	for _, key := range keys {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a slot
		go func(k string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the slot

			content, err := bucket.Download(k)
			if err != nil {
				errs <- fmt.Errorf("failed to download file %s: %w", k, err)
				return
			}

			results <- FileData{
				Key:     k,
				Content: content,
			}
		}(key)
	}

	// Step 4: Wait for all downloads to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
		close(errs)
	}()

	// Step 5: Handle the results and errors
	for {
		select {
		case file, ok := <-results:
			if !ok {
				results = nil
			} else {
				files = append(files, file)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
			} else {
				return nil, err
			}
		}

		if results == nil && errs == nil {
			break
		}
	}

	return files, nil
}
