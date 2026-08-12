package srvc

import (
	"context"
	"log/slog"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/sync/singleflight"
)

type TaskService interface {
	GetTask(ctx context.Context, shortId string) (Task, srvcerror.E)
	CreateTask(ctx context.Context, task Task) srvcerror.E
	DeleteTask(ctx context.Context, shortId string) srvcerror.E

	// test files
	UploadTestFile(ctx context.Context, body []byte) srvcerror.E
	GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, srvcerror.E)
	DownloadTestFile(ctx context.Context, testFileSha256 string) ([]byte, srvcerror.E)

	// illustration image
	UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, srvcerror.E)
	DeleteIllustrationImg(ctx context.Context, taskId string) srvcerror.E
	UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) srvcerror.E
	GetHttpUrlForIllustrImg(ctx context.Context, illstrImgS3Key string) (string, srvcerror.E)

	// markdown statement
	UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) srvcerror.E

	// markdown statement image
	UploadStatementImage(ctx context.Context, taskId string, filename string, mimeType string, body []byte) (string, srvcerror.E)
	DeleteStatementImage(ctx context.Context, taskId string, filename string) srvcerror.E
	GetHttpUrlForStatementImage(ctx context.Context, statementImageS3Key string) (string, srvcerror.E)

	// website
	GetTaskPreview(ctx context.Context, shortId string) (TaskPreview, srvcerror.E)
	ListTaskPreviews(ctx context.Context) ([]TaskPreview, srvcerror.E)

	// taskzip archive format
	ImportTaskFromZip(ctx context.Context, zipBytes []byte, overrideId string) (string, srvcerror.E)
	ExportTaskAsZip(ctx context.Context, taskId string) ([]byte, srvcerror.E)

	// unorganized
	GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, srvcerror.E)
	ResolveNames(ctx context.Context, shortIds []string) ([]string, srvcerror.E)
	SearchTasksByName(ctx context.Context, name string) ([]string, srvcerror.E)
}

type ObjectStore interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	Download(key string) ([]byte, error)
	Exists(key string) (bool, error)
	Delete(key string) error
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

type taskSrvc struct {
	publicStore   ObjectStore
	testfileStore ObjectStore

	repo TaskPgRepo

	testCache *TestFileCache

	apiPublicBaseURL           string
	testfileDownloadSigningKey []byte

	// dlGroup deduplicates concurrent DownloadTestFile calls per key to prevent thundering herd
	dlGroup singleflight.Group
}

type TaskSrvcOption func(*taskSrvc)

func WithPublicAPIBaseURL(baseURL string) TaskSrvcOption {
	return func(ts *taskSrvc) {
		ts.apiPublicBaseURL = baseURL
	}
}

func WithTestfileDownloadSigningKey(key []byte) TaskSrvcOption {
	return func(ts *taskSrvc) {
		ts.testfileDownloadSigningKey = key
	}
}

func NewTaskSrvc(
	repo TaskPgRepo,
	publicStore, testfileStore ObjectStore,
	opts ...TaskSrvcOption,
) *taskSrvc {
	ts := &taskSrvc{
		publicStore:                publicStore,
		testfileStore:              testfileStore,
		repo:                       repo,
		testCache:                  NewTestFileCache(),
		apiPublicBaseURL:           "http://localhost:8080",
		testfileDownloadSigningKey: []byte("testfile-download-signing-key-for-tests"),
	}
	for _, opt := range opts {
		opt(ts)
	}
	return ts
}

func (ts *taskSrvc) logger(ctx context.Context) *slog.Logger {
	return ctxlog.FromContext(ctx).With("module", "task", "layer", "srvc")
}
