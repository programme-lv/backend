package subm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/guregu/dynamo/v2"
)

// SubmissionRow represents the user data structure.
type SubmissionRow struct {
	Uuid       string `dynamo:"uuid,hash"` // Primary key
	UnixTime   int64  `dynamo:"unix_timestamp"`
	Content    string `dynamo:"content"`
	Version    int64  `dynamo:"version"` // For optimistic locking
	AuthorUuid string `dynamo:"author_uuid"`
	ProgLangId string `dynamo:"prog_lang_id"`
	TaskId     string `dynamo:"task_id"`
}

// DynamoDbSubmTable represents the DynamoDB table.
type DynamoDbSubmTable struct {
	ddbClient *dynamodb.Client
	tableName string
	submTable *dynamo.Table
}

// NewDynamoDbSubmTable initializes a new DynamoDbUsersTable.
func NewDynamoDbSubmTable(ddbClient *dynamodb.Client, tableName string) *DynamoDbSubmTable {
	ddb := &DynamoDbSubmTable{
		ddbClient: ddbClient,
		tableName: tableName,
	}
	db := dynamo.NewFromIface(ddb.ddbClient)
	table := db.Table(ddb.tableName)
	ddb.submTable = &table

	return ddb
}

func (ddb *DynamoDbSubmTable) Save(ctx context.Context, subm *SubmissionRow) error {
	// Increment the version number for optimistic locking
	subm.Version++

	put := ddb.submTable.Put(subm).If("attribute_not_exists(version) OR version = ?", subm.Version-1)
	return put.Run(ctx)
}

func (ddb *DynamoDbSubmTable) List(ctx context.Context) ([]*SubmissionRow, error) {
	var submissions []*SubmissionRow
	err := ddb.submTable.Scan().All(ctx, &submissions)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}
