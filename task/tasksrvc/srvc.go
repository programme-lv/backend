package tasksrvc

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/s3bucket"
	"github.com/programme-lv/backend/task/taskdomain"
	"github.com/programme-lv/backend/task/taskpgrepo"
)

type TaskSrvcClient interface {
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
	UploadStatementPdf(ctx context.Context, body []byte) (string, error)
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadMarkdownImage(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadTestFile(ctx context.Context, body []byte) error
	PutTask(ctx context.Context, task *taskdomain.Task) error
	GetTask(ctx context.Context, shortId string) (taskdomain.Task, error)
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, error)
	ListTasks(ctx context.Context) ([]taskdomain.Task, error)
}

type S3BucketFacade interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	PresignedURL(key string, duration time.Duration) (string, error)
	Exists(key string) (bool, error)
	ListAndGetAllFiles(prefix string) ([]s3bucket.FileData, error)
}

type TaskPgRepo interface {
	GetTask(ctx context.Context, shortId string) (taskdomain.Task, error)
	ListTasks(ctx context.Context, limit int, offset int) ([]taskdomain.Task, error)
	ResolveNames(ctx context.Context, shortIds []string) ([]string, error)
	Exists(ctx context.Context, shortId string) (bool, error)
}

type TaskSrvc struct {
	s3PublicBucket   S3BucketFacade
	s3TestfileBucket S3BucketFacade
	s3TaskBucket     S3BucketFacade

	repo TaskPgRepo
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(testFileSha256, time.Hour*24)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedUrl, nil
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

	repo := taskpgrepo.NewTaskPgRepo(pg)

	return &TaskSrvc{
		s3PublicBucket:   publicS3,
		s3TestfileBucket: testS3,
		s3TaskBucket:     taskS3,
		repo:             repo,
	}, nil
}
