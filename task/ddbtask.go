package task

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/guregu/dynamo/v2"
)

// TaskRow represents the user data structure.
type TaskRow struct {
	Id           string `dynamo:"PublishedID,hash"` // Primary key
	TomlManifest string `dynamo:"Manifest"`
	Version      int    `dynamo:"version"` // For optimistic locking
}

// DynamoDbTaskTable represents the DynamoDB table.
type DynamoDbTaskTable struct {
	ddbClient *dynamodb.Client
	tableName string
	taskTable *dynamo.Table
}

// NewDynamoDbTaskTable initializes a new DynamoDbUsersTable.
func NewDynamoDbTaskTable(ddbClient *dynamodb.Client, tableName string) *DynamoDbTaskTable {
	ddb := &DynamoDbTaskTable{
		ddbClient: ddbClient,
		tableName: tableName,
	}
	db := dynamo.NewFromIface(ddb.ddbClient)
	table := db.Table(ddb.tableName)
	ddb.taskTable = &table

	return ddb
}

// Get retrieves a task by ID from the DynamoDB table.
// Returns nil if the task is not found.
func (ddb *DynamoDbTaskTable) Get(ctx context.Context, id string) (*TaskRow, error) {
	user := new(TaskRow)

	err := ddb.taskTable.Get("PublishedID", id).One(ctx, user)
	if err != nil {
		if errors.Is(err, dynamo.ErrNotFound) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}

func (ddb *DynamoDbTaskTable) List(ctx context.Context) ([]*TaskRow, error) {
	var users []*TaskRow
	err := ddb.taskTable.Scan().All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Save saves a user to the DynamoDB table with optimistic locking.
func (ddb *DynamoDbTaskTable) Save(ctx context.Context, user *TaskRow) error {
	// Increment the version number for optimistic locking
	user.Version++

	put := ddb.taskTable.Put(user).If("attribute_not_exists(version) OR version = ?", user.Version-1)
	return put.Run(ctx)
}
