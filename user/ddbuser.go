package user

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
)

// UserRow represents the user data structure.
type UserRow struct {
	Uuid      string    `dynamo:"uuid,hash"` // Primary key
	Username  string    `dynamo:"username"`
	Email     string    `dynamo:"email"`
	BcryptPwd []byte    `dynamo:"bcrypt_pwd"`
	Firstname *string   `dynamo:"firstname"`
	Lastname  *string   `dynamo:"lastname"`
	Version   int       `dynamo:"version"` // For optimistic locking
	CreatedAt time.Time `dynamo:"created_at"`
}

// DynamoDbUserTable represents the DynamoDB table.
type DynamoDbUserTable struct {
	ddbClient  *dynamodb.Client
	tableName  string
	usersTable *dynamo.Table
}

// NewDynamoDbUsersTable initializes a new DynamoDbUsersTable.
func NewDynamoDbUsersTable(ddbClient *dynamodb.Client, tableName string) *DynamoDbUserTable {
	ddb := &DynamoDbUserTable{
		ddbClient: ddbClient,
		tableName: tableName,
	}
	db := dynamo.NewFromIface(ddb.ddbClient)
	table := db.Table(ddb.tableName)
	ddb.usersTable = &table

	return ddb
}

// Get retrieves a user by ID from the DynamoDB table.
func (ddb *DynamoDbUserTable) Get(ctx context.Context, uuid uuid.UUID) (*UserRow, error) {
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

func (ddb *DynamoDbUserTable) List(ctx context.Context) ([]*UserRow, error) {
	var users []*UserRow
	err := ddb.usersTable.Scan().All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Save saves a user to the DynamoDB table with optimistic locking.
func (ddb *DynamoDbUserTable) Save(ctx context.Context, user *UserRow) error {
	// Increment the version number for optimistic locking
	user.Version++

	put := ddb.usersTable.Put(user).If("attribute_not_exists(version) OR version = ?", user.Version-1)
	return put.Run(ctx)
}
