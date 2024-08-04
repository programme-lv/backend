package task

/*
package ddbtaskrepo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pelletier/go-toml/v2"
	"github.com/programme-lv/tasks-microservice/internal/domain"
)

type dynamoDbTaskRepo struct {
	db        *dynamodb.Client
	taskTable string
}

// ListTasks implements service.TaskRepo.
func (r *dynamoDbTaskRepo) ListTasks() ([]domain.Task, error) {
	response, err := r.db.Scan(context.Background(), &dynamodb.ScanInput{
		TableName: aws.String(r.taskTable),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %v", err)
	}

	tasks := []domain.Task{}
	for _, item := range response.Items {
		type TaskRow struct {
			PublishedID string `dynamodbav:"PublishedID"`
			Manifest    string `dynamodbav:"Manifest"`
		}
		row := TaskRow{}
		err = attributevalue.UnmarshalMap(item, &row)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task: %v", err)
		}

		tomlManifest := TaskTomlManifest{}
		err = toml.Unmarshal([]byte(row.Manifest), &tomlManifest)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task manifest: %v", err)
		}

		task, err := constructTaskFromManifest(row.PublishedID, &tomlManifest)
		if err != nil {
			return nil, fmt.Errorf("failed to construct task: %v", err)
		}

		tasks = append(tasks, *task)
	}

	return tasks, nil
}

func NewDynamoDbTaskRepo(db *dynamodb.Client, taskTable string) *dynamoDbTaskRepo {
	return &dynamoDbTaskRepo{
		db:        db,
		taskTable: taskTable,
	}
}

func (r *dynamoDbTaskRepo) GetTask(id string) (*domain.Task, error) {
	response, err := r.db.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"PublishedID": &types.AttributeValueMemberS{Value: id},
		},
		TableName: aws.String(r.taskTable),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %v", err)
	}

	type TaskRow struct {
		PublishedID string `dynamodbav:"PublishedID"`
		Manifest    string `dynamodbav:"Manifest"`
	}
	row := TaskRow{}
	err = attributevalue.UnmarshalMap(response.Item, &row)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %v", err)
	}

	tomlManifest := TaskTomlManifest{}
	err = toml.Unmarshal([]byte(row.Manifest), &tomlManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %v", err)
	}

	task, err := constructTaskFromManifest(row.PublishedID, &tomlManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to construct task: %v", err)
	}

	return task, nil
}
*/
