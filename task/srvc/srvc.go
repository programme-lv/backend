package srvc

import (
	"context"
	"fmt"
	"time"

	"github.com/programme-lv/backend/s3bucket"
)

type TaskSrvcClient interface {
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
	UploadStatementPdf(ctx context.Context, body []byte) (string, error)
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error)
	UploadStatementImage(ctx context.Context, taskId string, semanticFilename string, mimeType string, body []byte) (string, error)
	DeleteStatementImage(ctx context.Context, taskId string, s3Uri string) error
	UploadTestFile(ctx context.Context, body []byte) error
	GetTask(ctx context.Context, shortId string) (Task, error)
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, error)
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, task Task) error
	ResolveNames(ctx context.Context, shortIds []string) ([]string, error)
	UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) error
}

type S3BucketFacade interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	PresignedURL(key string, duration time.Duration) (string, error)
	Exists(key string) (bool, error)
	ListAndGetAllFiles(prefix string) ([]s3bucket.FileData, error)
	Delete(key string) error
	Bucket() string
}

type TaskPgRepo interface {
	GetTask(ctx context.Context, shortId string) (Task, error)
	ListTasks(ctx context.Context, limit int, offset int) ([]Task, error)
	ResolveNames(ctx context.Context, shortIds []string) ([]string, error)
	Exists(ctx context.Context, shortId string) (bool, error)
	CreateTask(ctx context.Context, task Task) error
	UpdateStatement(ctx context.Context, taskId string, statement MarkdownStatement) error
	AddStatementImg(ctx context.Context, taskId string, img StatementImage) error
	DeleteStatementImg(ctx context.Context, taskId string, s3Uri string) error
}

type TaskSrvc struct {
	s3PublicBucket   S3BucketFacade
	s3TestfileBucket S3BucketFacade

	repo TaskPgRepo
}

// ResolveNames implements TaskSrvcClient.
func (ts *TaskSrvc) ResolveNames(ctx context.Context, shortIds []string) ([]string, error) {
	names, err := ts.repo.ResolveNames(ctx, shortIds)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve names: %w", err)
	}
	return names, nil
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(fmt.Sprintf("%s.zst", testFileSha256), time.Hour*24)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedUrl, nil
}

func NewTaskSrvc(repo TaskPgRepo, publicS3, testS3 *s3bucket.S3Bucket) (TaskSrvcClient, error) {
	return &TaskSrvc{
		s3PublicBucket:   publicS3,
		s3TestfileBucket: testS3,
		repo:             repo,
	}, nil
}
