package srvc

import (
	"context"
	"fmt"
	"time"
)

func (ts *TaskSrvc) SearchTasksByName(ctx context.Context, name string) ([]string, error) {
	taskIds, err := ts.repo.SearchTasksByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return taskIds, nil
}

func (ts *TaskSrvc) GetTaskPreview(ctx context.Context, id string) (res TaskPreview, err error) {
	exists, err := ts.repo.Exists(ctx, id)
	if err != nil {
		return TaskPreview{}, err
	}
	if !exists {
		return TaskPreview{}, NewErrorTaskNotFound(id)
	}
	taskPreview, err := ts.repo.GetTaskPreview(ctx, id)
	if err != nil {
		return TaskPreview{}, err
	}
	return taskPreview, nil
}

func (ts *TaskSrvc) ListTaskPreviews(ctx context.Context) ([]TaskPreview, error) {
	taskPreviews, err := ts.repo.ListTaskPreviews(ctx, 100, 0)
	if err != nil {
		ts.logger(ctx).Error("failed to list task previews", "error", err)
		return nil, NewErrorInternalServerError()
	}
	return taskPreviews, nil
}

func (ts *TaskSrvc) GetTask(ctx context.Context, id string) (res Task, err error) {
	exists, err := ts.repo.Exists(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if !exists {
		return Task{}, NewErrorTaskNotFound(id)
	}
	task, err := ts.repo.GetTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func (ts *TaskSrvc) ListTasks(ctx context.Context) ([]Task, error) {
	tasks, err := ts.repo.ListTasks(ctx, 100, 0)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (ts *TaskSrvc) GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, error) {
	fullNames, err := ts.repo.ResolveNames(ctx, shortIDs)
	if err != nil {
		return nil, err
	}
	return fullNames, nil
}

func (ts *TaskSrvc) GetPublicUrlForIllustrImg(ctx context.Context, s3Key string) (string, error) {
	if ts.s3PublicBucket.Bucket() == "proglv-public" {
		cloudfrontEndpoint := "https://dvhk4hiwp1rmf.cloudfront.net/"
		return cloudfrontEndpoint + s3Key, nil
	} else {
		return ts.s3PublicBucket.PresignedURL(s3Key, 24*time.Hour)
	}
}

func (ts *TaskSrvc) GetPublicUrlForStatementImage(ctx context.Context, s3Key string) (string, error) {
	if ts.s3PublicBucket.Bucket() == "proglv-public" {
		cloudfrontEndpoint := "https://dvhk4hiwp1rmf.cloudfront.net/"
		return cloudfrontEndpoint + s3Key, nil
	} else {
		return ts.s3PublicBucket.PresignedURL(s3Key, 24*time.Hour)
	}
}

// GetTestDownlUrl implements submadapter.TaskSrvcFacade.
func (ts *TaskSrvc) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	presignedUrl, err := ts.s3TestfileBucket.PresignedURL(fmt.Sprintf("%s.zst", testFileSha256), time.Hour*24)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedUrl, nil
}

// ResolveNames implements TaskSrvcClient.
func (ts *TaskSrvc) ResolveNames(ctx context.Context, shortIds []string) ([]string, error) {
	names, err := ts.repo.ResolveNames(ctx, shortIds)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve names: %w", err)
	}
	if len(names) != len(shortIds) {
		return nil, fmt.Errorf("expected %d names, got %d", len(shortIds), len(names))
	}
	return names, nil
}
