package tasksrvc

import (
	"context"
)

func (ts *TaskService) GetTask(ctx context.Context, id string) (res Task, err error) {
	for _, task := range ts.tasks {
		if task.ShortId == id {
			res = task
			// replaceUuidsWithValidUrls(&res)
			return res, nil
		}
	}
	return res, NewErrorTaskNotFound(id)
}

func (ts *TaskService) ListTasks(ctx context.Context) ([]Task, error) {
	tasks := make([]Task, 0, len(ts.tasks))
	tasks = append(tasks, ts.tasks...)
	return tasks, nil
}

func (ts *TaskService) GetTaskFullNames(ctx context.Context, shortIDs []string) ([]string, error) {
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

// func replaceUuidsWithValidUrls(task *Task) {
// 	for i, statement := range task.MdStatements {
// 		f := func(s string) string {
// 			for _, img := range statement.Images {
// 				s = strings.ReplaceAll(s, img.Uuid, img.S3Url)
// 			}
// 			s = strings.ReplaceAll(s, "https://proglv-public.s3.eu-central-1.amazonaws.com/", PublicCloudfrontEndpoint)
// 			return s
// 		}
// 		task.MdStatements[i].Story = f(statement.Story)
// 		task.MdStatements[i].Input = f(statement.Input)
// 		task.MdStatements[i].Output = f(statement.Output)
// 		task.MdStatements[i].Notes = f(statement.Notes)
// 		task.MdStatements[i].Scoring = f(statement.Scoring)
// 	}
// }
