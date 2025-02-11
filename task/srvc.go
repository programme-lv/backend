package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/programme-lv/backend/s3bucket"
)

type TaskSrvcClient interface {
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context) ([]Task, error)
	GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, error)
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
	UploadStatementPdf(ctx context.Context, body []byte) (string, error)
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadMarkdownImage(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadTestFile(ctx context.Context, body []byte) error
	PutTask(ctx context.Context, task *Task) error
}

type TaskSrvc struct {
	tasks []Task

	s3PublicBucket   *s3bucket.S3Bucket
	s3TestfileBucket *s3bucket.S3Bucket
	s3TaskBucket     *s3bucket.S3Bucket
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(testFileSha256, time.Hour*24)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedUrl, nil
}

func NewTaskSrvc() (*TaskSrvc, error) {
	publicBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-public")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	testFileBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	taskBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tasks")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}

	start := time.Now()
	slog.Info("downloading tasks from S3", "bucket", taskBucket.Bucket())
	taskFiles, err := taskBucket.ListAndGetAllFiles("")
	if err != nil {
		return nil, fmt.Errorf("failed to list task files: %w", err)
	}
	elapsed := time.Since(start)
	slog.Info("downloaded tasks from S3",
		"bucket", taskBucket.Bucket(),
		"count", len(taskFiles),
		"time_ms", elapsed.Milliseconds())

	// unmarshall jsons in taskFiles to tasks
	tasks := []Task{}
	for _, taskFile := range taskFiles {
		task := Task{}
		err = json.Unmarshal(taskFile.Content, &task)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return &TaskSrvc{
		tasks: tasks,

		s3PublicBucket:   publicBucket,
		s3TestfileBucket: testFileBucket,
		s3TaskBucket:     taskBucket,
	}, nil
}
