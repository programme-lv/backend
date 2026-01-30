package srvc

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/programme-lv/backend/common/srvcerror"
)

func (ts *TaskSrvc) SearchTasksByName(ctx context.Context, name string) ([]string, srvcerror.E) {
	taskIds, err := ts.repo.SearchTasksByName(ctx, name)
	if err != nil {
		ts.logger(ctx).Error("search tasks by name", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return taskIds, nil
}

func (ts *TaskSrvc) GetTaskPreview(ctx context.Context, id string) (res TaskPreview, err srvcerror.E) {
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
	return taskPreview, nil
}

func (ts *TaskSrvc) ListTaskPreviews(ctx context.Context) ([]TaskPreview, srvcerror.E) {
	taskPreviews, err := ts.repo.ListTaskPreviews(ctx, 100, 0)
	if err != nil {
		ts.logger(ctx).Error("list task previews", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return taskPreviews, nil
}

//go:embed embedded/it-task-note.md
var itTaskNote string

func (ts *TaskSrvc) GetTask(ctx context.Context, id string) (Task, srvcerror.E) {
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

func (ts *TaskSrvc) ListTasks(ctx context.Context) ([]Task, srvcerror.E) {
	tasks, err := ts.repo.ListTasks(ctx, 100, 0)
	if err != nil {
		ts.logger(ctx).Error("list tasks", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return tasks, nil
}

func (ts *TaskSrvc) GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, srvcerror.E) {
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

func (ts *TaskSrvc) GetHttpUrlForIllustrImg(ctx context.Context, s3Key string) (string, srvcerror.E) {
	if ts.s3PublicBucket.Bucket() == "proglv-public" {
		cloudfrontEndpoint := "https://dvhk4hiwp1rmf.cloudfront.net/"
		return cloudfrontEndpoint + s3Key, nil
	} else {
		url, err := ts.s3PublicBucket.PresignedURL(s3Key, 24*time.Hour)
		if err != nil {
			ts.logger(ctx).Error("presign illustrate image", "error", err)
			return "", NewErrorInternalServerError()
		}
		return url, nil
	}
}

func (ts *TaskSrvc) GetHttpUrlForStatementImage(ctx context.Context, s3Key string) (string, srvcerror.E) {
	if ts.s3PublicBucket.Bucket() == "proglv-public" {
		cloudfrontEndpoint := "https://dvhk4hiwp1rmf.cloudfront.net/"
		return cloudfrontEndpoint + s3Key, nil
	} else {
		url, err := ts.s3PublicBucket.PresignedURL(s3Key, 24*time.Hour)
		if err != nil {
			ts.logger(ctx).Error("presign statement image", "error", err)
			return "", NewErrorInternalServerError()
		}
		return url, nil
	}
}

func (ts *TaskSrvc) GetHttpUrlForPdfStatement(ctx context.Context, s3Key string) (string, srvcerror.E) {
	if ts.s3PublicBucket.Bucket() == "proglv-public" {
		cloudfrontEndpoint := "https://dvhk4hiwp1rmf.cloudfront.net/"
		return cloudfrontEndpoint + s3Key, nil
	} else {
		url, err := ts.s3PublicBucket.PresignedURL(s3Key, 24*time.Hour)
		if err != nil {
			ts.logger(ctx).Error("presign pdf statement", "error", err)
			return "", NewErrorInternalServerError()
		}
		return url, nil
	}
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, srvcerror.E) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(fmt.Sprintf("%s.zst", testFileSha256), time.Hour*24)
	if err != nil {
		ts.logger(ctx).Error("presign test file", "error", err)
		return "", NewErrorInternalServerError()
	}
	return presignedUrl, nil
}

// ResolveNames implements TaskSrvcClient.
func (ts *TaskSrvc) ResolveNames(ctx context.Context, shortIds []string) ([]string, srvcerror.E) {
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
