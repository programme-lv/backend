package tasksrvc

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// dynamodb task details row
type ddbDetailsRow struct {
	TaskCode    string  `dynamodbav:"task_code"`
	FullName    string  `dynamodbav:"full_name"`
	MemMbytes   int     `dynamodbav:"mem_mbytes"`
	CpuSecs     float64 `dynamodbav:"cpu_secs"`
	Difficulty  *int    `dynamodbav:"difficulty"`
	OriginOlymp string  `dynamodbav:"origin_olymp"`
	IllustrKey  *string `dynamodbav:"illustr_key"`
}

func (row ddbDetailsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: "details#"},
	}
}

// dynamodb visible input subtasks row
type ddbVisInpStsRow struct {
	TaskCode string `dynamodbav:"task_code"`
	Subtask  int    `dynamodbav:"subtask"`
	TestId   int    `dynamodbav:"test_id"`
	Input    string `dynamodbav:"input"`
}

func (row ddbVisInpStsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.Subtask == 0 || row.TestId == 0 {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("vis_inp_sts#%d#%d", row.Subtask, row.TestId)},
	}
}

type ddbItemStruct interface {
	GetKey() map[string]types.AttributeValue
}

// marshalDdbItem marshals the item and includes its key attributes.
func marshalDdbItem(item ddbItemStruct) map[string]types.AttributeValue {
	marshalled, err := attributevalue.MarshalMap(item)
	if err != nil {
		panic(err)
	}
	// Merge the key attributes into the marshalled map
	maps.Copy(marshalled, item.GetKey())
	return marshalled
}

// PutItem inserts an item into the DynamoDB table.
func (ts *TaskService) PutItem(item ddbItemStruct) error {
	_, err := ts.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &ts.taskTableName,
		Item:      marshalDdbItem(item),
	})
	return err
}

func (ts *TaskService) PutItems(ctx context.Context, items ...ddbItemStruct) error {
	const batchSize = 25 // DynamoDB BatchWriteItem limit

	if len(items) == 0 {
		log.Println("No items provided for batch put.")
		return nil
	}

	// Prepare the write requests
	writeRequests := make([]types.WriteRequest, 0, len(items))
	for _, item := range items {
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: marshalDdbItem(item),
			},
		})
	}

	// Batch the write requests
	for i := 0; i < len(writeRequests); i += batchSize {
		end := i + batchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}
		batch := writeRequests[i:end]

		// Create the BatchWriteItem input
		batchInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				ts.taskTableName: batch,
			},
		}

		// Execute the batch write with retry logic
		err := ts.batchWriteWithRetry(ctx, batchInput, 5)
		if err != nil {
			return fmt.Errorf("failed to batch write items: %w", err)
		}

		log.Printf("Successfully put items %d to %d.", i+1, end)
	}

	return nil
}

func (ts *TaskService) batchWriteWithRetry(ctx context.Context, batchInput *dynamodb.BatchWriteItemInput, maxRetries int) error {
	var err error
	currentRetry := 0
	for {
		var resp *dynamodb.BatchWriteItemOutput
		resp, err = ts.ddbClient.BatchWriteItem(ctx, batchInput)
		if err != nil {
			return err
		}

		// Check for unprocessed items
		if len(resp.UnprocessedItems) == 0 {
			// All items processed successfully
			return nil
		}

		// If there are unprocessed items, prepare for retry
		unprocessed, exists := resp.UnprocessedItems[ts.taskTableName]
		if !exists || len(unprocessed) == 0 {
			// No unprocessed items to retry
			return nil
		}

		if currentRetry >= maxRetries {
			return fmt.Errorf("max retries reached with %d unprocessed items", len(unprocessed))
		}

		// Exponential backoff before retrying
		backoffDuration := time.Duration(100*(1<<currentRetry)) * time.Millisecond
		log.Printf("Retrying %d unprocessed items after %v...", len(unprocessed), backoffDuration)
		time.Sleep(backoffDuration)

		// Update the batchInput with unprocessed items for the next retry
		batchInput.RequestItems[ts.taskTableName] = unprocessed
		currentRetry++
	}
}
