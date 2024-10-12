package tasksrvc

import (
	"context"
	"strings"
)

const PublicCloudfrontEndpoint = "https://dvhk4hiwp1rmf.cloudfront.net/"

func (ts *TaskService) GetTask(ctx context.Context, id string) (res Task, err error) {
	for _, task := range ts.tasks {
		if task.ShortId == id {
			res = task
			replaceUuidsWithValidUrls(&res)
			return res, nil
		}
	}
	return res, NewErrorTaskNotFound(id)
}

func (ts *TaskService) ListTasks(ctx context.Context) (tasks []Task, err error) {
	tasks = []Task{}
	for _, task := range ts.tasks {
		replaceUuidsWithValidUrls(&task)
		tasks = append(tasks, task)
	}
	return ts.tasks, nil
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

func replaceUuidsWithValidUrls(task *Task) {
	for i, statement := range task.MdStatements {
		f := func(s string) string {
			return replaceUuids(task.AssetUuidToUrl, s)
		}
		task.MdStatements[i].Story = f(statement.Story)
		task.MdStatements[i].Input = f(statement.Input)
		task.MdStatements[i].Output = f(statement.Output)
		task.MdStatements[i].Notes = f(statement.Notes)
		task.MdStatements[i].Scoring = f(statement.Scoring)
	}
}

func replaceUuids(uuidToUrl map[string]string, text string) string {
	for k, v := range uuidToUrl {
		text = strings.ReplaceAll(text, k, v)
	}
	text = strings.ReplaceAll(text, "https://proglv-public.s3.eu-central-1.amazonaws.com/", PublicCloudfrontEndpoint)
	return text
}
