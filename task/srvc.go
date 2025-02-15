package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/s3bucket"
)

type TaskSrvcClient interface {
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
	UploadStatementPdf(ctx context.Context, body []byte) (string, error)
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadMarkdownImage(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadTestFile(ctx context.Context, body []byte) error
	PutTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, shortId string) (Task, error)
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, error)
	ListTasks(ctx context.Context) ([]Task, error)
}

type S3BucketFacade interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	PresignedURL(key string, duration time.Duration) (string, error)
	Exists(key string) (bool, error)
	ListAndGetAllFiles(prefix string) ([]s3bucket.FileData, error)
}

type TaskSrvc struct {
	tasks []Task

	s3PublicBucket   S3BucketFacade
	s3TestfileBucket S3BucketFacade
	s3TaskBucket     S3BucketFacade

	pg *pgxpool.Pool
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(testFileSha256, time.Hour*24)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedUrl, nil
}

func NewTaskSrvc(pg *pgxpool.Pool, publicS3, testS3, taskS3 S3BucketFacade) (TaskSrvcClient, error) {
	start := time.Now()
	taskFiles, err := taskS3.ListAndGetAllFiles("")
	if err != nil {
		return nil, fmt.Errorf("failed to list task files: %w", err)
	}
	elapsed := time.Since(start)
	slog.Info("downloaded tasks from S3",
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

		s3PublicBucket:   publicS3,
		s3TestfileBucket: testS3,
		s3TaskBucket:     taskS3,
	}, nil
}

func NewDefaultTaskSrvc() (TaskSrvcClient, error) {
	publicS3, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-public")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	testS3, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	taskS3, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tasks")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}

	pg, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to create pg pool: %w", err)
	}

	return NewTaskSrvc(pg, publicS3, testS3, taskS3)
}
