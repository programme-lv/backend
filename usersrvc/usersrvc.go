package usersrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	// assuming custom Latvian translations
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type UserService struct {
	postgres *sqlx.DB
}

func NewUsers() *UserService {
	postgresConnStr := getPostgresConnStr()
	db, err := sqlx.Connect("postgres", postgresConnStr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}
	return &UserService{
		postgres: db,
	}
}

func getPostgresConnStr() string {
	user := os.Getenv("POSTGRES_USER")
	secretName := os.Getenv("POSTGRES_PASSWORD_SECRET_NAME")
	secretValue, err := getSecretFromAWS(secretName)
	if err != nil {
		panic(fmt.Sprintf("failed to get postgres password from AWS: %v", err))
	}
	var secret struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(secretValue), &secret); err != nil {
		panic(fmt.Sprintf("failed to parse postgres password secret: %v", err))
	}
	pw := secret.Password
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pw, db, ssl)
}

func getSecretFromAWS(secretName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}
	svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", err
	}
	return *result.SecretString, nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (res *User, err error) {
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return nil, newErrInternalSE().SetDebug(errMsg)
	}

	var resSlice []User = make([]User, 0)
	for _, user := range allUsers {
		if user.Username == username {
			if len(resSlice) == 1 {
				format := "multiple users with the same username: %s"
				errMsg := fmt.Errorf(format, username)
				return nil, newErrInternalSE().SetDebug(errMsg)
			}

			genUser := User{
				UUID:      user.UUID,
				Username:  user.Username,
				Email:     user.Email,
				Firstname: &user.Firstname,
				Lastname:  &user.Lastname,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		format := "user with username %s not found"
		errMsg := fmt.Errorf(format, username)
		errRes := newErrUserNotFound().SetDebug(errMsg)
		return nil, errRes
	}

	return &resSlice[0], nil
}

func (s *UserService) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (res *User, err error) {
	// Start Generation Here
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return nil, newErrInternalSE().SetDebug(errMsg)
	}

	var resSlice []User
	for _, user := range allUsers {
		if user.UUID == uuid {
			if len(resSlice) == 1 {
				format := "multiple users with the same UUID: %s"
				errMsg := fmt.Errorf(format, uuid)
				return nil, newErrInternalSE().SetDebug(errMsg)
			}

			genUser := User{
				UUID:      user.UUID,
				Username:  user.Username,
				Email:     user.Email,
				Firstname: &user.Firstname,
				Lastname:  &user.Lastname,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		format := "user with UUID %s not found"
		errMsg := fmt.Errorf(format, uuid)
		errRes := newErrUserNotFound().SetDebug(errMsg)
		return nil, errRes
	}

	return &resSlice[0], nil
}

func (s *UserService) GetUsernames(ctx context.Context,
	uuids []uuid.UUID) ([]string, error) {

	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return nil, newErrInternalSE().SetDebug(errMsg)
	}

	usernames := make([]string, 0, len(uuids))

	for _, id := range uuids {
		found := false
		for _, user := range allUsers {
			if user.UUID == id {
				usernames = append(usernames, user.Username)
				found = true
				break
			}
		}
		if !found {
			return nil, newErrUserNotFound()
		}
	}

	return usernames, nil
}
