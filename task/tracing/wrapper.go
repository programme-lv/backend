package tracing

import (
	"context"
	"fmt"

	"github.com/programme-lv/backend/task/srvc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TaskSrvcTracer wraps a TaskSrvcClient with tracing capabilities
type TaskSrvcTracer struct {
	client srvc.TaskSrvcClient
	tracer trace.Tracer
}

// NewTaskSrvcTracer creates a new TaskSrvcTracer
func NewTaskSrvcTracer(client srvc.TaskSrvcClient) *TaskSrvcTracer {
	return &TaskSrvcTracer{
		client: client,
		tracer: otel.Tracer("task-service"),
	}
}

// GetTestDownlUrl implements TaskSrvcClient.GetTestDownlUrl with tracing
func (t *TaskSrvcTracer) GetTestDownlUrl(ctx context.Context, testFileSha256 string) (string, error) {
	ctx, span := t.tracer.Start(ctx, "GetTestDownlUrl")
	defer span.End()

	span.SetAttributes(attribute.String("test_file_sha256", testFileSha256))

	url, err := t.client.GetTestDownlUrl(ctx, testFileSha256)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetAttributes(attribute.String("url", url))
	return url, nil
}

// UploadStatementPdf implements TaskSrvcClient.UploadStatementPdf with tracing
func (t *TaskSrvcTracer) UploadStatementPdf(ctx context.Context, body []byte) (string, error) {
	ctx, span := t.tracer.Start(ctx, "UploadStatementPdf")
	defer span.End()

	span.SetAttributes(attribute.Int("body_size", len(body)))

	url, err := t.client.UploadStatementPdf(ctx, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetAttributes(attribute.String("url", url))
	return url, nil
}

// UploadIllustrationImg implements TaskSrvcClient.UploadIllustrationImg with tracing
func (t *TaskSrvcTracer) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, error) {
	ctx, span := t.tracer.Start(ctx, "UploadIllustrationImg")
	defer span.End()

	span.SetAttributes(
		attribute.String("mime_type", mimeType),
		attribute.Int("body_size", len(body)),
	)

	url, err := t.client.UploadIllustrationImg(ctx, mimeType, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetAttributes(attribute.String("url", url))
	return url, nil
}

// UploadMarkdownImage implements TaskSrvcClient.UploadMarkdownImage with tracing
func (t *TaskSrvcTracer) UploadMarkdownImage(ctx context.Context, mimeType string, body []byte) (string, error) {
	ctx, span := t.tracer.Start(ctx, "UploadMarkdownImage")
	defer span.End()

	span.SetAttributes(
		attribute.String("mime_type", mimeType),
		attribute.Int("body_size", len(body)),
	)

	url, err := t.client.UploadMarkdownImage(ctx, mimeType, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetAttributes(attribute.String("url", url))
	return url, nil
}

// UploadTestFile implements TaskSrvcClient.UploadTestFile with tracing
func (t *TaskSrvcTracer) UploadTestFile(ctx context.Context, body []byte) error {
	ctx, span := t.tracer.Start(ctx, "UploadTestFile")
	defer span.End()

	span.SetAttributes(attribute.Int("body_size", len(body)))

	err := t.client.UploadTestFile(ctx, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// GetTask implements TaskSrvcClient.GetTask with tracing
func (t *TaskSrvcTracer) GetTask(ctx context.Context, shortId string) (srvc.Task, error) {
	ctx, span := t.tracer.Start(ctx, "GetTask")
	defer span.End()

	span.SetAttributes(attribute.String("short_id", shortId))

	task, err := t.client.GetTask(ctx, shortId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return srvc.Task{}, err
	}

	span.SetAttributes(
		attribute.String("task_name", task.FullName),
		attribute.Int("num_tests", len(task.Tests)),
		attribute.Int("num_subtasks", len(task.Subtasks)),
	)

	return task, nil
}

// GetTaskFullNames implements TaskSrvcClient.GetTaskFullNames with tracing
func (t *TaskSrvcTracer) GetTaskFullNames(ctx context.Context, shortIds []string) ([]string, error) {
	ctx, span := t.tracer.Start(ctx, "GetTaskFullNames")
	defer span.End()

	span.SetAttributes(
		attribute.Int("num_short_ids", len(shortIds)),
		attribute.StringSlice("short_ids", shortIds),
	)

	names, err := t.client.GetTaskFullNames(ctx, shortIds)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("num_names_found", len(names)))

	return names, nil
}

// ListTasks implements TaskSrvcClient.ListTasks with tracing
func (t *TaskSrvcTracer) ListTasks(ctx context.Context) ([]srvc.Task, error) {
	ctx, span := t.tracer.Start(ctx, "ListTasks")
	defer span.End()

	tasks, err := t.client.ListTasks(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("num_tasks", len(tasks)))

	return tasks, nil
}

// CreateTask implements TaskSrvcClient.CreateTask with tracing
func (t *TaskSrvcTracer) CreateTask(ctx context.Context, task srvc.Task) error {
	ctx, span := t.tracer.Start(ctx, "CreateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("short_id", task.ShortId),
		attribute.String("full_name", task.FullName),
		attribute.Int("num_tests", len(task.Tests)),
		attribute.Int("num_subtasks", len(task.Subtasks)),
	)

	err := t.client.CreateTask(ctx, task)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to create task: %v", err))
		return err
	}

	return nil
}
