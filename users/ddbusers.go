package users

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
)

// UserRow represents the user data structure.
type UserRow struct {
	Uuid      string  `dynamo:"uuid,hash"` // Primary key
	Username  string  `dynamo:"username"`
	Email     string  `dynamo:"email"`
	BcryptPwd string  `dynamo:"bcrypt_pwd"`
	Firstname *string `dynamo:"firstname"`
	Lastname  *string `dynamo:"lastname"`
	Version   int     `dynamo:"version"` // For optimistic locking
}

// DynamoDbUsersTable represents the DynamoDB table.
type DynamoDbUsersTable struct {
	ddbClient  *dynamodb.Client
	tableName  string
	usersTable dynamo.Table
}

// NewDynamoDbUsersTable initializes a new DynamoDbUsersTable.
func NewDynamoDbUsersTable(ddbClient *dynamodb.Client, tableName string) *DynamoDbUsersTable {
	ddb := &DynamoDbUsersTable{
		ddbClient: ddbClient,
		tableName: tableName,
	}
	db := dynamo.NewFromIface(ddb.ddbClient)
	table := db.Table(ddb.tableName)
	ddb.usersTable = table

	return ddb
}

// Get retrieves a user by ID from the DynamoDB table.
func (ddb *DynamoDbUsersTable) Get(ctx context.Context, uuid uuid.UUID) (*UserRow, error) {
	user := new(UserRow)

	id := uuid.String()

	err := ddb.usersTable.Get("uuid", id).One(ctx, user)
	if err != nil {
		if errors.Is(err, dynamo.ErrNotFound) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}

// Save saves a user to the DynamoDB table with optimistic locking.
func (ddb *DynamoDbUsersTable) Save(ctx context.Context, user *UserRow) error {
	// Increment the version number for optimistic locking
	user.Version++

	put := ddb.usersTable.Put(user).If("attribute_not_exists(version) OR version = ?", user.Version-1)
	return put.Run(ctx)
}
