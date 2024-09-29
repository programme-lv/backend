package tasksrvc

import (
	"context"
)

const PublicCloudfrontEndpoint = "https://dvhk4hiwp1rmf.cloudfront.net/"

func (ts *TaskService) GetTask(ctx context.Context, id string) (task *Task, err error) {
	panic("not implemented")
}

func (ts *TaskService) ListTasks(ctx context.Context) (tasks []*Task, err error) {
	panic("not implemented")
}
