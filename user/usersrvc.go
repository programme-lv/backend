package user

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb" // assuming custom Latvian translations
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	ddbUserTable *DynamoDbUserTable
}

func NewUsers() *UserService {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	userTableName := os.Getenv("DDB_USER_TABLE_NAME")
	if userTableName == "" {
		slog.Error("DDB_USER_TABLE_NAME is not set")
		os.Exit(1)
	}

	return &UserService{
		ddbUserTable: NewDynamoDbUsersTable(dynamodbClient, userTableName),
	}
}

func (s *UserService) Login(ctx context.Context, p *LoginPayload) (res *User, err error) {
	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			err = bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(p.Password))
			if err == nil {
				return &User{
					UUID:      user.Uuid,
					Username:  user.Username,
					Email:     user.Email,
					Firstname: user.Firstname,
					Lastname:  user.Lastname,
				}, nil
			}
		}
	}

	return nil, newErrUsernameOrPasswordIncorrect()
}

func (s *UserService) GetUserByUsername(ctx context.Context, p *GetUserByUsernamePayload) (res *User, err error) {
	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	var resSlice []User = make([]User, 0)
	for _, user := range allUsers {
		if user.Username == p.Username {
			if len(resSlice) == 1 {
				return nil, fmt.Errorf("multiple users with the same username")
			}

			genUser := User{
				UUID:      user.Uuid,
				Username:  user.Username,
				Email:     user.Email,
				Firstname: user.Firstname,
				Lastname:  user.Lastname,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		errRes := newErrUserNotFound()
		errRes.SetDebugInfo(fmt.Errorf("user with username %s not found", p.Username))
		return nil, errRes
	}

	return &resSlice[0], nil
}
