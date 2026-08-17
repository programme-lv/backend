package srvc

import (
	"context"

	"github.com/programme-lv/backend/common/srvcerror"
)

// CreateTask stores statement-image keys without the md-images/ prefix.
func (ts *taskSrvc) CreateTask(ctx context.Context, task Task) srvcerror.E {
	for i := range task.MdImages {
		task.MdImages[i].ObjectKey = taskStatementImageStoredKey(task.MdImages[i].ObjectKey)
	}
	err := ts.repo.CreateTask(ctx, task)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("create task", "error", err)
		return srvcerror.InternalServerError()
	}
	return nil
}

// DeleteTask removes database rows for the task. Object-store files are not deleted.
func (ts *taskSrvc) DeleteTask(ctx context.Context, shortId string) srvcerror.E {
	l := ts.logger(ctx)
	err := ts.repo.DeleteTask(ctx, shortId)
	if err != nil {
		l.Error("delete task", "task_id", shortId, "error", err)
		return srvcerror.InternalServerError()
	}
	l.Info("task deleted successfully", "task_id", shortId)
	return nil
}

func (ts *taskSrvc) UpdateStatementMd(ctx context.Context, taskId string, statement MarkdownStatement) srvcerror.E {
	err := ts.repo.UpdateStatement(ctx, taskId, statement)
	if err != nil {
		l := ts.logger(ctx)
		l.Error("update statement", "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}
