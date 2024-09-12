package task

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

type s3Bucket struct {
	client *s3.Client
	bucket string
}

func NewS3BucketUploader(region string, bucket string) *s3Bucket {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	return &s3Bucket{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}
}

func (bucket *s3Bucket) Upload(content []byte, key string, mediaType string) error {
	_, err := bucket.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &bucket.bucket,
		Key:         &key,
		Body:        bytes.NewReader(content),
		ContentType: &mediaType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %v", err)
	}
	return nil
}

func (bucket *s3Bucket) Exists(key string) (bool, error) {
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
		return false, fmt.Errorf("failed to check object existence: %v", err)
	}
	return true, nil
}

func (bucket *s3Bucket) Download(key string) ([]byte, error) {
	output, err := bucket.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %v", err)
	}
	defer output.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	return buf.Bytes(), nil
}
