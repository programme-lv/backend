package srvc

import (
	"context"
)

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
