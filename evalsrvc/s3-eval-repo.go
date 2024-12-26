package evalsrvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3EvalRepo struct {
	client     *s3.Client
	bucketName string
}

func NewS3EvalRepo(client *s3.Client, bucketName string) *S3EvalRepo {
	return &S3EvalRepo{
		client:     client,
		bucketName: bucketName,
	}
}

func (r *S3EvalRepo) Save(ctx context.Context, eval Evaluation) error {
	data, err := json.Marshal(eval)
	if err != nil {
		return fmt.Errorf("failed to marshal evaluation: %w", err)
	}

	key := fmt.Sprintf("%s.json", eval.UUID.String())
	fmt.Printf("Saving with key: %s\n", key)
	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to store evaluation in S3: %w", err)
	}

	return nil
}

func (r *S3EvalRepo) Get(ctx context.Context, evalUuid uuid.UUID) (*Evaluation, error) {
	key := fmt.Sprintf("%s.json", evalUuid.String())
	fmt.Printf("Getting with key: %s\n", key)

	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("S3 Get error: %v\n", err)
		return nil, fmt.Errorf("failed to get evaluation from S3: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation data: %w", err)
	}

	var eval Evaluation
	if err := json.Unmarshal(data, &eval); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evaluation: %w", err)
	}

	return &eval, nil
}
