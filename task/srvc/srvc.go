package srvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/s3bucket"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/sync/singleflight"
)

type TaskSrvcClient interface {
	GetTask(ctx context.Context, shortId string) (Task, *srvcerror.Error)
	ListTasks(ctx context.Context) ([]Task, *srvcerror.Error)
	CreateTask(ctx context.Context, task Task) *srvcerror.Error
	DeleteTask(ctx context.Context, shortId string) *srvcerror.Error

	// test files
	UploadTestFile(ctx context.Context, body []byte) *srvcerror.Error
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, *srvcerror.Error)
	DownloadTestFile(ctx context.Context, testFileSha256 string) ([]byte, *srvcerror.Error)

	// original pdf statement
	UploadStatementPdf(ctx context.Context, body []byte) (string, *srvcerror.Error)
	GetHttpUrlForPdfStatement(ctx context.Context, pdfStatementS3Key string) (string, *srvcerror.Error)

	// illustration image
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, *srvcerror.Error)
	DeleteIllustrationImg(ctx context.Context, taskId string) *srvcerror.Error
	UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) *srvcerror.Error
	GetHttpUrlForIllustrImg(ctx context.Context, illstrImgS3Key string) (string, *srvcerror.Error)

	// markdown statement
	UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) *srvcerror.Error

	// markdown statement image
	UploadStatementImage(ctx context.Context, taskId string, filename string, mimeType string, body []byte) (string, *srvcerror.Error)
	DeleteStatementImage(ctx context.Context, taskId string, filename string) *srvcerror.Error
	GetHttpUrlForStatementImage(ctx context.Context, statementImageS3Key string) (string, *srvcerror.Error)

	// website
	GetTaskPreview(ctx context.Context, shortId string) (TaskPreview, *srvcerror.Error)
	ListTaskPreviews(ctx context.Context) ([]TaskPreview, *srvcerror.Error)

	// taskzip archive format
	ImportTaskFromZip(ctx context.Context, zipBytes []byte, overrideId string) (string, *srvcerror.Error)
	ExportTaskAsZip(ctx context.Context, taskId string) ([]byte, *srvcerror.Error)

	// unorganized
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, *srvcerror.Error)
	ResolveNames(ctx context.Context, shortIds []string) ([]string, *srvcerror.Error)
	SearchTasksByName(ctx context.Context, name string) ([]string, *srvcerror.Error)

	// original file archive
	UploadOgFileArchive(ctx context.Context, zipBytes []byte) (string, *srvcerror.Error)
	DownloadOgFileArchive(ctx context.Context, s3Key string) ([]byte, *srvcerror.Error)
}

type S3BucketFacade interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	Download(key string) ([]byte, error)
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
	DeleteTask(ctx context.Context, shortId string) error
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

	// dlGroup deduplicates concurrent DownloadTestFile calls per key to prevent thundering herd
	dlGroup singleflight.Group
}

func NewTaskSrvc(
	repo TaskPgRepo,
	cdnS3, testS3 S3BucketFacade,
) (TaskSrvcClient, error) {
	return &TaskSrvc{
		s3PublicBucket:   cdnS3,
		s3TestfileBucket: testS3,
		repo:             repo,
		testCache:        NewTestFileCache(),
	}, nil
}

func (ts *TaskSrvc) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "srvc")
}
