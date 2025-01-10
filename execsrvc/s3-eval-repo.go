package execsrvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3EvalRepo struct {
	logger     *slog.Logger
	client     *s3.Client
	bucketName string
}

func NewS3ExecRepo(logger *slog.Logger, client *s3.Client, bucketName string) *S3EvalRepo {
	return &S3EvalRepo{
		logger:     logger,
		client:     client,
		bucketName: bucketName,
	}
}

func (r *S3EvalRepo) Save(ctx context.Context, eval *Execution) error {
	data, err := json.Marshal(eval)
	if err != nil {
		return fmt.Errorf("failed to marshal evaluation: %w", err)
	}

	key := fmt.Sprintf("%s.json", eval.UUID.String())
	r.logger.Info("saving eval to S3", "key", key)

	// add additional timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		r.logger.Error("failed to store evaluation in S3", "error", err)
		return fmt.Errorf("failed to store evaluation in S3: %w", err)
	}

	return nil
}

func (r *S3EvalRepo) Get(ctx context.Context, evalUuid uuid.UUID) (*Execution, error) {
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

	var eval Execution
	if err := json.Unmarshal(data, &eval); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evaluation: %w", err)
	}

	return &eval, nil
}
