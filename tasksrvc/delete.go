package tasksrvc

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DeleteTask deletes all items associated with the given taskCode.
func (ts *TaskService) DeleteTask(taskCode string) error {
	ctx := context.TODO()
	pkValue := fmt.Sprintf("task#%s", taskCode)

	// Query all items with the specified partition key
	items, err := ts.queryItemsByPartitionKey(ctx, pkValue)
	if err != nil {
		return fmt.Errorf("failed to query items for deletion: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	// Extract keys from items
	var keysToDelete []map[string]types.AttributeValue
	for _, item := range items {
		key, exists := item["pk"]
		if !exists {
			continue
		}
		sk, exists := item["sk"]
		if !exists {
			continue
		}

		keysToDelete = append(keysToDelete, map[string]types.AttributeValue{
			"pk": key,
			"sk": sk,
		})
	}

	// Batch delete items
	err = ts.deleteItemsBatch(ctx, keysToDelete)
	if err != nil {
		return fmt.Errorf("failed to delete items: %w", err)
	}

	log.Println("All matching items have been deleted.")
	return nil
}

// deleteItemsBatch deletes a batch of items given their keys.
func (ts *TaskService) deleteItemsBatch(ctx context.Context, keys []map[string]types.AttributeValue) error {
	const batchSize = 25
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]

		writeRequests := make([]types.WriteRequest, 0, len(batch))
		for _, key := range batch {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: key,
				},
			})
		}

		batchWriteInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				ts.taskTableName: writeRequests,
			},
		}

		_, err := ts.ddbClient.BatchWriteItem(ctx, batchWriteInput)
		if err != nil {
			return fmt.Errorf("failed to batch write items: %w", err)
		}
	}

	return nil
}
