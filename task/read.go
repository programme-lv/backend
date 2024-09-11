package task

import "context"

func (ts *TaskService) GetTask(ctx context.Context, taskId string) (task *Task, err error) {
	return nil, nil
}

func (ts *TaskService) ListTasks(ctx context.Context) (tasks []*Task, err error) {
	return nil, nil
}

func (ts *TaskService) GetTaskSubmEvalData(ctx context.Context,
	taskId string) (data *TaskSubmEvalData, err error) {
	return nil, nil
}
