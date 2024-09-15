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
	// Initialize the Scan input to retrieve all items from the table
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(ts.taskTableName),
	}

	// Create a paginator to handle large datasets efficiently
	paginator := dynamodb.NewScanPaginator(ts.ddbClient, scanInput)

	// Map to categorize items by their 'pk' value
	pkMap := make(map[string][]map[string]types.AttributeValue)

	// Iterate through each page of scan results
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan items: %w", err)
		}

		// Iterate through each item in the current page
		for _, item := range page.Items {
			// Extract the 'pk' attribute from the item
			pkAttr, ok := item["pk"].(*types.AttributeValueMemberS)
			if !ok {
				return nil, fmt.Errorf("item missing 'pk' or 'pk' is not a string: %v", item)
			}
			pk := pkAttr.Value

			// Append the item to the corresponding 'pk' group
			pkMap[pk] = append(pkMap[pk], item)
		}
	}

	// Iterate through each 'pk' group to construct Task objects
	for pk, items := range pkMap {
		constructor := newTaskConstructor()

		// Apply each item to the task constructor
		for _, item := range items {
			if err := constructor.applyDdbItem(item); err != nil {
				return nil, fmt.Errorf("failed to apply DynamoDB item for pk '%s': %w", pk, err)
			}
		}

		// Retrieve the constructed Task object
		task := constructor.getTask()
		tasks = append(tasks, task)
	}

	return tasks, nil
}
