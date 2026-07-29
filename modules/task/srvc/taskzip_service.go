package srvc

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/programme-lv/backend/common/srvcerror"
	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

func (ts *taskSrvc) ImportTaskFromZip(
	ctx context.Context, zipBytes []byte, overrideID string,
) (string, srvcerror.E) {
	archive, err := taskzipv1.Read(zipBytes)
	if err != nil {
		return "", taskZipReadError(err)
	}
	task, err := mapFromTaskZip(archive, overrideID)
	if err != nil {
		return "", NewErrorInvalidTaskZip(err.Error())
	}
	if err := prepareTaskZipImages(archive, &task); err != nil {
		return "", NewErrorInvalidTaskZip(err.Error())
	}
	exists, err := ts.repo.Exists(ctx, task.ShortId)
	if err != nil {
		ts.logger(ctx).Error("check task existence", "error", err)
		return "", NewErrorInternalServerError()
	}
	if exists {
		return "", NewErrorTaskAlreadyExists(task.ShortId)
	}
	if err := ts.uploadTaskZipAssets(ctx, archive, &task); err != nil {
		return "", err
	}
	if err := ts.repo.CreateTask(ctx, task); err != nil {
		ts.logger(ctx).Error("create task", "error", err)
		return "", NewErrorInternalServerError()
	}
	return task.ShortId, nil
}

func taskZipReadError(err error) srvcerror.E {
	if errors.Is(err, taskzipv1.ErrInteractive) || errors.Is(err, taskzipv1.ErrAttached) {
		return NewErrorUnsupportedTaskZip(err.Error())
	}
	return NewErrorInvalidTaskZip(err.Error())
}

func prepareTaskZipImages(archive taskzipv1.Task, task *Task) error {
	for name, data := range archive.StatementImages {
		mimeType, err := taskZipImageMime(name)
		if err != nil {
			return fmt.Errorf("statement image %s: unsupported type", name)
		}
		width, height := 0, 0
		if mimeType == "image/png" || mimeType == "image/jpeg" || mimeType == "image/jpg" {
			width, height, err = getImgWidthHeighPx(data, mimeType)
			if err != nil {
				return fmt.Errorf("statement image %s: %w", name, err)
			}
		}
		task.MdImages = append(task.MdImages, StatementImage{
			S3Key:    fmt.Sprintf("%s/%s%s", task.ShortId, Sha2Hex(data)[:12], filepath.Ext(name)),
			Filename: name, WidthPx: width, HeightPx: height, SzInBytes: len(data),
		})
	}
	return nil
}

func (ts *taskSrvc) uploadTaskZipAssets(
	ctx context.Context, archive taskzipv1.Task, task *Task,
) srvcerror.E {
	for _, image := range task.MdImages {
		data := archive.StatementImages[image.Filename]
		mimeType, _ := taskZipImageMime(image.Filename)
		if _, err := ts.publicStore.Upload(
			data, taskStatementImageObjectKey(image.S3Key), mimeType,
		); err != nil {
			ts.logger(ctx).Error("upload statement image", "error", err)
			return NewErrorInternalServerError()
		}
	}
	task.Tests = make([]Test, len(archive.Tests))
	for i, test := range archive.Tests {
		if err := ts.uploadTaskZipTest(ctx, i, test, task); err != nil {
			return err
		}
	}
	return nil
}

func taskZipImageMime(name string) (string, error) {
	if mimeType, err := MimeFromFname(name); err == nil && mimeType != "" {
		return mimeType, nil
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	default:
		return "", fmt.Errorf("unsupported image type")
	}
}

func (ts *taskSrvc) uploadTaskZipTest(
	ctx context.Context, i int, test taskzipv1.Test, task *Task,
) srvcerror.E {
	if err := ts.UploadTestFile(ctx, test.Input); err != nil {
		return err
	}
	if err := ts.UploadTestFile(ctx, test.Output); err != nil {
		return err
	}
	task.Tests[i] = Test{InpSha2: Sha2Hex(test.Input), AnsSha2: Sha2Hex(test.Output)}
	return nil
}

func (ts *taskSrvc) ExportTaskAsZip(ctx context.Context, taskID string) ([]byte, srvcerror.E) {
	task, getErr := ts.GetTask(ctx, taskID)
	if getErr != nil {
		return nil, getErr
	}
	archive, err := mapToTaskZip(task)
	if err != nil {
		ts.logger(ctx).Error("map TaskZip", "task_id", taskID, "error", err)
		return nil, NewErrorInternalServerError()
	}
	if err := ts.downloadTaskZipAssets(ctx, task, &archive); err != nil {
		ts.logger(ctx).Error("download TaskZip assets", "task_id", taskID, "error", err)
		return nil, NewErrorInternalServerError()
	}
	data, err := taskzipv1.Write(archive)
	if err != nil {
		ts.logger(ctx).Error("write TaskZip", "task_id", taskID, "error", err)
		return nil, NewErrorInternalServerError()
	}
	return data, nil
}

func (ts *taskSrvc) downloadTaskZipAssets(
	ctx context.Context, task Task, archive *taskzipv1.Task,
) error {
	for _, image := range task.MdImages {
		data, err := ts.publicStore.Download(taskStatementImageObjectKey(image.S3Key))
		if err != nil {
			ts.logger(ctx).Error("download statement image", "error", err)
			return err
		}
		archive.StatementImages[image.Filename] = data
	}
	for _, test := range task.Tests {
		input, err := ts.DownloadTestFile(ctx, test.InpSha2)
		if err != nil {
			return fmt.Errorf("download test input: %w", err)
		}
		output, err := ts.DownloadTestFile(ctx, test.AnsSha2)
		if err != nil {
			return fmt.Errorf("download test output: %w", err)
		}
		archive.Tests = append(archive.Tests, taskzipv1.Test{Input: input, Output: output})
	}
	return nil
}
