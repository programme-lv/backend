package s3bucket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Bucket struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3Bucket(region string, bucket string) (*S3Bucket, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &S3Bucket{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		region: region,
	}, nil
}

// Upload uploads the given content to the S3 bucket with the specified key and media type.
// It returns the URL of the uploaded object or an error if the upload fails.
//
// Parameters:
//   - content: The byte slice containing the content to be uploaded.
//   - key: The key (path) under which the content will be stored in the S3 bucket.
//   - mediaType: The MIME type of the content being uploaded.
//
// Returns:
//   - string: The URL of the uploaded object.
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

	// Construct the Object URL
	objectURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket.bucket, bucket.region, key)

	return objectURL, nil
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
	buf.ReadFrom(output.Body)
	return buf.Bytes(), nil
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
