package user

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb" // assuming custom Latvian translations
	"github.com/programme-lv/backend/auth"
	"goa.design/clue/log"
	"golang.org/x/crypto/bcrypt"
)

type UsersSrvc struct {
	jwtKey       []byte
	ddbUserTable *DynamoDbUserTable
}

func NewUsers(ctx context.Context) *UsersSrvc {
	// read jwt key from env
	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatalf(ctx,
			errors.New("JWT_KEY is not set"),
			"cant read JWT_KEY from env in new user service constructor")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
		config.WithLogger(log.AsAWSLogger(ctx)))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	userTableName := os.Getenv("DDB_USER_TABLE_NAME")
	if userTableName == "" {
		log.Fatalf(ctx,
			errors.New("DDB_USER_TABLE_NAME is not set"),
			"cant read DDB_USER_TABLE_NAME from env in new user service constructor")
	}

	return &UsersSrvc{
		jwtKey: []byte(jwtKey),
		ddbUserTable: NewDynamoDbUsersTable(
			dynamodbClient, userTableName),
	}
}

// User login
func (s *UsersSrvc) Login(ctx context.Context, p *LoginPayload) (res string, err error) {
	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return "", fmt.Errorf("error listing users: %w", err)
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			err = bcrypt.CompareHashAndPassword(user.BcryptPwd, []byte(p.Password))
			if err == nil {
				token, err := auth.GenerateJWT(
					user.Username,
					user.Email, user.Uuid,
					user.Firstname, user.Lastname,
					s.jwtKey)
				if err != nil {
					return "", fmt.Errorf("error generating JWT: %w", err)
				}
				if token == "" {
					return "", fmt.Errorf("error generating JWT")
				}
				return token, nil
			}
		}
	}

	return "", newErrUsernameOrPasswordIncorrect()
}

// GetUserByUsername implements users.Service.
func (s *UsersSrvc) GetUserByUsername(ctx context.Context, p *GetUserByUsernamePayload) (res *User, err error) {
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

			firstname := ""
			if user.Firstname != nil {
				firstname = *user.Firstname
			}

			lastname := ""
			if user.Lastname != nil {
				lastname = *user.Lastname
			}

			genUser := User{
				UUID:      user.Uuid,
				Username:  user.Username,
				Email:     user.Email,
				Firstname: firstname,
				Lastname:  lastname,
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
