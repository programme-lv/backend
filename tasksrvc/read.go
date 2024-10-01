package tasksrvc

import (
	"context"
)

const PublicCloudfrontEndpoint = "https://dvhk4hiwp1rmf.cloudfront.net/"

func (ts *TaskService) GetTask(ctx context.Context, id string) (task *Task, err error) {
	for _, task := range ts.tasks {
		if task.ShortId == id {
			return &task, nil
		}
	}
	return nil, NewErrorTaskNotFound(id)
}

func (ts *TaskService) ListTasks(ctx context.Context) (tasks []Task, err error) {
	return ts.tasks, nil
}
