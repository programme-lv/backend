package usersrvc

import (
	"context"
	"fmt"
	"os"

	// assuming custom Latvian translations
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	pw := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pw, db, ssl)
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
