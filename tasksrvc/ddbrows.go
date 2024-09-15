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
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("vis_inp_sts#%02d#%03d", row.Subtask, row.TestId)},
	}
}

type ddbTestGroupsRow struct {
	TaskCode string `dynamodbav:"task_code"`
	GroupId  int    `dynamodbav:"group_id"`
	Points   int    `dynamodbav:"points"`
	Public   bool   `dynamodbav:"public"`
	Subtask  int    `dynamodbav:"subtask"`
	TestIds  []int  `dynamodbav:"test_ids"`
}

func (row ddbTestGroupsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.GroupId == 0 {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("test_groups#%02d", row.GroupId)},
	}
}

type ddbTestChsumsRow struct {
	TaskCode string `dynamodbav:"task_code"`
	TestId   int    `dynamodbav:"test_id"`
	InSha2   string `dynamodbav:"in_sha2"`
	AnsSha2  string `dynamodbav:"ans_sha2"`
}

func (row ddbTestChsumsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.TestId == 0 {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("test_chsums#%03d", row.TestId)},
	}
}

type ddbPdfSttmentsRow struct {
	TaskCode string `dynamodbav:"task_code"`
	LangIso  string `dynamodbav:"lang_iso639"`
	PdfSha2  string `dynamodbav:"pdf_sha2"`
}

func (row ddbPdfSttmentsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.LangIso == "" {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("pdf_sttments#%s", row.LangIso)},
	}
}

type ddbMdSttmentsRow struct {
	TaskCode string  `dynamodbav:"task_code"`
	LangIso  string  `dynamodbav:"lang_iso639"`
	Story    string  `dynamodbav:"story"`
	Input    string  `dynamodbav:"input"`
	Output   string  `dynamodbav:"output"`
	Scoring  *string `dynamodbav:"scoring"`
	Notes    *string `dynamodbav:"notes"`
}

func (row ddbMdSttmentsRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.LangIso == "" {
		return nil
	}

	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("md_sttments#%s", row.LangIso)},
	}
}

type ddbImgUuidMapRow struct {
	TaskCode string `dynamodbav:"task_code"`
	Uuid     string `dynamodbav:"uuid"`
	S3Key    string `dynamodbav:"s3_key"`
}

func (row ddbImgUuidMapRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.Uuid == "" {
		return nil
	}

	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("img_uuid_map#%s", row.Uuid)},
	}
}

type ddbExamplesRow struct {
	TaskCode  string `dynamodbav:"task_code"`
	ExampleId int    `dynamodbav:"example_id"`
	Input     string `dynamodbav:"input"`
	Output    string `dynamodbav:"output"`
	MdNote    string `dynamodbav:"md_note"`
}

func (row ddbExamplesRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.ExampleId == 0 {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("examples#%03d", row.ExampleId)},
	}
}

type ddbOriginNotesRow struct {
	TaskCode string `dynamodbav:"task_code"`
	LangIso  string `dynamodbav:"lang_iso639"`
	OgInfo   string `dynamodbav:"og_info"`
}

func (row ddbOriginNotesRow) GetKey() map[string]types.AttributeValue {
	if row.TaskCode == "" || row.LangIso == "" {
		return nil
	}
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("task#%s", row.TaskCode)},
		"sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("origin_notes#%s", row.LangIso)},
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
