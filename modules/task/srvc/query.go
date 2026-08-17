package srvc

import (
	"context"
	_ "embed"
	"time"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/srvcerror"
)

const testfileDownloadURLValidity = time.Hour

func (ts *taskSrvc) SearchTasksByName(ctx context.Context, name string) ([]string, srvcerror.E) {
	taskIds, err := ts.repo.SearchTasksByName(ctx, name)
	if err != nil {
		ts.logger(ctx).Error("search tasks by name", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return taskIds, nil
}

func (ts *taskSrvc) GetTaskPreview(ctx context.Context, id string) (res TaskPreview, err srvcerror.E) {
	exists, errDb := ts.repo.Exists(ctx, id)
	if errDb != nil {
		ts.logger(ctx).Error("exists task preview", "error", errDb)
		return TaskPreview{}, NewErrorInternalServerError()
	}
	if !exists {
		return TaskPreview{}, NewErrorTaskNotFound(id)
	}
	taskPreview, errDb := ts.repo.GetTaskPreview(ctx, id)
	if errDb != nil {
		ts.logger(ctx).Error("get task preview", "error", errDb)
		return TaskPreview{}, NewErrorInternalServerError()
	}
	applyPreviewOriginNotes(&taskPreview)
	return taskPreview, nil
}

func (ts *taskSrvc) ListTaskPreviews(ctx context.Context) ([]TaskPreview, srvcerror.E) {
	taskPreviews, err := ts.repo.ListTaskPreviews(ctx, 100, 0)
	if err != nil {
		ts.logger(ctx).Error("list task previews", "error", err)
		return nil, NewErrorInternalServerError()
	}
	for i := range taskPreviews {
		applyPreviewOriginNotes(&taskPreviews[i])
	}
	return taskPreviews, nil
}

//go:embed embedded/it-task-note.md
var itTaskNote string

func (ts *taskSrvc) GetTask(ctx context.Context, id string) (Task, srvcerror.E) {
	exists, err := ts.repo.Exists(ctx, id)
	if err != nil {
		ts.logger(ctx).Error("check if task exists", "error", err)
		return Task{}, NewErrorInternalServerError()
	}
	if !exists {
		return Task{}, NewErrorTaskNotFound(id)
	}
	task, err := ts.repo.GetTask(ctx, id)
	if err != nil {
		ts.logger(ctx).Error("get task", "error", err)
		return Task{}, NewErrorInternalServerError()
	}
	applyOriginNotes(&task)
	if task.Interactor != "" {
		for i := range task.MdStatements {
			if task.MdStatements[i].Notes == "" {
				task.MdStatements[i].Notes = itTaskNote
			}
			// task.MdStatements[i].Notes = itTaskNote
		}
	}
	return task, nil
}

func (ts *taskSrvc) GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, srvcerror.E) {
	fullNames, err := ts.repo.ResolveNames(ctx, shortIDs)
	if err != nil {
		ts.logger(ctx).Error("resolve names (full names)", "error", err)
		return nil, NewErrorInternalServerError()
	}
	if len(fullNames) != len(shortIDs) {
		return nil, ErrSomeTaskNotFound
	}
	return fullNames, nil
}

func (ts *taskSrvc) GetHttpUrlForIllustrImg(ctx context.Context, objectKey string) (string, srvcerror.E) {
	url, err := filestore.AssetURL(ts.apiPublicBaseURL, taskIllustrationObjectKey(objectKey))
	if err != nil {
		ts.logger(ctx).Error("build illustration image URL", "error", err)
		return "", NewErrorInternalServerError()
	}
	return url, nil
}

func (ts *taskSrvc) GetHttpUrlForStatementImage(ctx context.Context, objectKey string) (string, srvcerror.E) {
	url, err := filestore.AssetURL(ts.apiPublicBaseURL, taskStatementImageObjectKey(objectKey))
	if err != nil {
		ts.logger(ctx).Error("build statement image URL", "error", err)
		return "", NewErrorInternalServerError()
	}
	return url, nil
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *taskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, srvcerror.E) {
	return filestore.SignedTestfileURL(
		ts.apiPublicBaseURL,
		testFileSha256,
		ts.testfileDownloadSigningKey,
		time.Now().Add(testfileDownloadURLValidity),
	), nil
}

// ResolveNames implements TaskSrvcClient.
func (ts *taskSrvc) ResolveNames(ctx context.Context, shortIds []string) ([]string, srvcerror.E) {
	names, err := ts.repo.ResolveNames(ctx, shortIds)
	if err != nil {
		ts.logger(ctx).Error("resolve names", "error", err)
		return nil, NewErrorInternalServerError()
	}
	if len(names) != len(shortIds) {
		return nil, ErrSomeTaskNotFound
	}
	return names, nil
}
