package task

import (
	"context"
)

func (ts *TaskSrvc) GetTask(ctx context.Context, id string) (res Task, err error) {
	for _, task := range ts.tasks {
		if task.ShortId == id {
			res = task
			return res, nil
		}
	}
	return res, NewErrorTaskNotFound(id)
}

func (ts *TaskSrvc) ListTasks(ctx context.Context) ([]Task, error) {
	tasks := make([]Task, 0, len(ts.tasks))
	tasks = append(tasks, ts.tasks...)
	return tasks, nil
}

func (ts *TaskSrvc) GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, error) {
	fullNames := make([]string, 0, len(shortIDs))

	for _, shortID := range shortIDs {
		found := false
		for _, task := range ts.tasks {
			if task.ShortId == shortID {
				fullNames = append(fullNames, task.FullName)
				found = true
				break
			}
		}
		if !found {
			return nil, NewErrorTaskNotFound(shortID)
		}
	}

	return fullNames, nil
}
