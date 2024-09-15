package tasksrvc

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const publicCloudfrontURLPrefix = "https://dvhk4hiwp1rmf.cloudfront.net/"

func (ts *TaskService) GetTask(ctx context.Context, taskCode string) (task *Task, err error) {
	pkValue := fmt.Sprintf("task#%s", taskCode)

	queryInput := &dynamodb.QueryInput{
		TableName:                &ts.taskTableName,
		KeyConditionExpression:   aws.String("#pk = :pkval"),
		ExpressionAttributeNames: map[string]string{"#pk": "pk"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkval": &types.AttributeValueMemberS{Value: pkValue},
		},
	}

	var items []map[string]types.AttributeValue
	paginator := dynamodb.NewQueryPaginator(ts.ddbClient, queryInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query items: %w", err)
		}
		items = append(items, page.Items...)
	}

	if len(items) == 0 {
		return nil, newErrTaskNotFound()
	}

	constructor := newTaskConstructor()
	for _, item := range items {
		if err := constructor.applyDdbItem(item); err != nil {
			return nil, fmt.Errorf("failed to apply ddb item: %w", err)
		}
	}

	return constructor.getTask(), nil
}

func (ts *TaskService) ListTasks(ctx context.Context) (tasks []*Task, err error) {
	return nil, nil
}

func (ts *TaskService) GetTaskSubmEvalData(ctx context.Context,
	taskId string) (data *TaskSubmEvalData, err error) {
	return nil, nil
}
