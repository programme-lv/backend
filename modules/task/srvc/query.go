package srvc

import (
	"context"
	_ "embed"

	"github.com/programme-lv/backend/common/srvcerror"
)

func (ts *taskSrvc) SearchTasksByName(ctx context.Context, name string) ([]string, srvcerror.E) {
	taskIds, err := ts.repo.SearchTasksByName(ctx, name)
	if err != nil {
		ts.logger(ctx).Error("search tasks by name", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	return taskIds, nil
}

func (ts *taskSrvc) GetTaskPreview(ctx context.Context, id string) (res TaskPreview, err srvcerror.E) {
	exists, errDb := ts.repo.Exists(ctx, id)
	if errDb != nil {
		ts.logger(ctx).Error("exists task preview", "error", errDb)
		return TaskPreview{}, srvcerror.InternalServerError()
	}
	if !exists {
		return TaskPreview{}, errTaskNotFound(id)
	}
	taskPreview, errDb := ts.repo.GetTaskPreview(ctx, id)
	if errDb != nil {
		ts.logger(ctx).Error("get task preview", "error", errDb)
		return TaskPreview{}, srvcerror.InternalServerError()
	}
	applyPreviewOriginNotes(&taskPreview)
	return taskPreview, nil
}

// ListTaskPreviews returns up to 100 task previews.
func (ts *taskSrvc) ListTaskPreviews(ctx context.Context) ([]TaskPreview, srvcerror.E) {
	taskPreviews, err := ts.repo.ListTaskPreviews(ctx, 100, 0)
	if err != nil {
		ts.logger(ctx).Error("list task previews", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	for i := range taskPreviews {
		applyPreviewOriginNotes(&taskPreviews[i])
	}
	return taskPreviews, nil
}

// ListTaskFilters returns the origin catalog for the public task list.
// The tree is built from every task row, not the 100-preview list cap.
func (ts *taskSrvc) ListTaskFilters(ctx context.Context) (FilterTree, srvcerror.E) {
	rows, err := ts.repo.ListOriginCounts(ctx)
	if err != nil {
		ts.logger(ctx).Error("list origin counts", "error", err)
		return FilterTree{}, srvcerror.InternalServerError()
	}
	return BuildFilterTree(rows), nil
}

//go:embed embedded/it-task-note.md
var itTaskNote string

func (ts *taskSrvc) GetTask(ctx context.Context, id string) (Task, srvcerror.E) {
	exists, err := ts.repo.Exists(ctx, id)
	if err != nil {
		ts.logger(ctx).Error("check if task exists", "error", err)
		return Task{}, srvcerror.InternalServerError()
	}
	if !exists {
		return Task{}, errTaskNotFound(id)
	}
	task, err := ts.repo.GetTask(ctx, id)
	if err != nil {
		ts.logger(ctx).Error("get task", "error", err)
		return Task{}, srvcerror.InternalServerError()
	}
	applyOriginNotes(&task)
	if task.Interactor != "" {
		for i := range task.MdStatements {
			if task.MdStatements[i].Notes == "" {
				task.MdStatements[i].Notes = itTaskNote
			}
		}
	}
	return task, nil
}

func (ts *taskSrvc) ResolveNames(ctx context.Context, shortIds []string) ([]string, srvcerror.E) {
	names, err := ts.repo.ResolveNames(ctx, shortIds)
	if err != nil {
		ts.logger(ctx).Error("resolve names", "error", err)
		return nil, srvcerror.InternalServerError()
	}
	if len(names) != len(shortIds) {
		return nil, ErrSomeTaskNotFound
	}
	return names, nil
}
