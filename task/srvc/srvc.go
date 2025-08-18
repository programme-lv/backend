package srvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/s3bucket"
)

type TaskSrvcClient interface {
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error)
	UploadStatementPdf(ctx context.Context, body []byte) (string, error)
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error)
	DeleteIllustrationImg(ctx context.Context, taskId string) error
	UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) error
	UploadStatementImage(ctx context.Context, taskId string, filename string, mimeType string, body []byte) (string, error)
	DeleteStatementImage(ctx context.Context, taskId string, filename string) error
	UploadTestFile(ctx context.Context, body []byte) error
	GetTask(ctx context.Context, shortId string) (Task, error)
	GetTaskPreview(ctx context.Context, shortId string) (TaskPreview, error)
	ListTaskPreviews(ctx context.Context) ([]TaskPreview, error)
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, error)
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, task Task) error
	ResolveNames(ctx context.Context, shortIds []string) ([]string, error)
	SearchTasksByName(ctx context.Context, name string) ([]string, error)
	UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) error
	GetPublicUrlForIllustrImg(ctx context.Context, illstrImgS3Key string) (string, error)
	GetPublicUrlForStatementImage(ctx context.Context, statementImageS3Key string) (string, error)
	ExportTaskAsZip(ctx context.Context, taskId string) ([]byte, error)
	GetCacheStats() (totalSizeMB int64, fileCount int, err error)
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
	GetTaskPreview(ctx context.Context, shortId string) (TaskPreview, error)
	SearchTasksByName(ctx context.Context, name string) ([]string, error)
	ListTasks(ctx context.Context, limit int, offset int) ([]Task, error)
	ListTaskPreviews(ctx context.Context, limit int, offset int) ([]TaskPreview, error)
	ResolveNames(ctx context.Context, shortIds []string) ([]string, error)
	Exists(ctx context.Context, shortId string) (bool, error)
	CreateTask(ctx context.Context, task Task) error
	UpdateStatement(ctx context.Context, taskId string, statement MarkdownStatement) error
	AddStatementImg(ctx context.Context, taskId string, img StatementImage) error
	DeleteStatementImg(ctx context.Context, taskId string, filename string) error
	UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) error
}

type TaskSrvc struct {
	s3PublicBucket   S3BucketFacade
	s3TestfileBucket S3BucketFacade

	repo TaskPgRepo

	testCache *TestFileCache
}

func NewTaskSrvc(repo TaskPgRepo, publicS3, testfileS3 *s3bucket.S3Bucket) (TaskSrvcClient, error) {
	return &TaskSrvc{
		s3PublicBucket:   publicS3,
		s3TestfileBucket: testfileS3,
		repo:             repo,
		testCache:        NewTestFileCache(),
	}, nil
}

func (ts *TaskSrvc) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "srvc")
}
