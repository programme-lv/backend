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
	log.Printf("Creating S3 bucket uploader with region: %s, bucket: %s", region, bucket)

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
	log.Printf("Uploading to S3 bucket: %s, key: %s", bucket.bucket, key)
	_, err := bucket.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &bucket.bucket,
		Key:         &key,
		Body:        bytes.NewReader(content),
		ContentType: &mediaType,
	})
	if err != nil {
		log.Printf("Failed to upload object: %v", err)
		return fmt.Errorf("failed to upload object: %v", err)
	}
	log.Printf("Successfully uploaded object to S3 bucket: %s, key: %s", bucket.bucket, key)
	return nil
}

func (bucket *s3Bucket) Exists(key string) (bool, error) {
	log.Printf("Checking if key: %s exists in S3 bucket: %s", key, bucket.bucket)
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
		log.Fatalf("Failed to check object existence: %v", err)
		return false, fmt.Errorf("failed to check object existence: %v", err)
	}
	log.Printf("Key: %s exists in S3 bucket: %s", key, bucket.bucket)
	return true, nil
}

func (bucket *s3Bucket) Download(key string) ([]byte, error) {
	log.Printf("Downloading from S3 bucket: %s, key: %s", bucket.bucket, key)
	output, err := bucket.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket.bucket,
		Key:    &key,
	})
	if err != nil {
		log.Printf("Failed to download object: %v", err)
		return nil, fmt.Errorf("failed to download object: %v", err)
	}
	defer output.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	log.Printf("Successfully downloaded object from S3 bucket: %s, key: %s", bucket.bucket, key)
	return buf.Bytes(), nil
}
