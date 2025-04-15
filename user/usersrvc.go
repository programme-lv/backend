package user

import (
	"context"
	"fmt"

	// assuming custom Latvian translations
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/conf"
)

type UserSrvc struct {
	postgres *sqlx.DB
}

func NewUserService() *UserSrvc {
	postgresConnStr := conf.GetPgConnStrFromEnv()
	db, err := sqlx.Connect("postgres", postgresConnStr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}
	return &UserSrvc{
		postgres: db,
	}
}

func (s *UserSrvc) GetUserByUsername(ctx context.Context, username string) (res User, err error) {
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return User{}, newErrInternalSE().SetDebug(errMsg)
	}

	var resSlice []User = make([]User, 0)
	for _, user := range allUsers {
		if user.Username == username {
			if len(resSlice) == 1 {
				format := "multiple users with the same username: %s"
				errMsg := fmt.Errorf(format, username)
				return User{}, newErrInternalSE().SetDebug(errMsg)
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
		return User{}, errRes
	}

	return resSlice[0], nil
}

func (s *UserSrvc) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (res User, err error) {
	// Start Generation Here
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return User{}, newErrInternalSE().SetDebug(errMsg)
	}

	var resSlice []User
	for _, user := range allUsers {
		if user.UUID == uuid {
			if len(resSlice) == 1 {
				format := "multiple users with the same UUID: %s"
				errMsg := fmt.Errorf(format, uuid)
				return User{}, newErrInternalSE().SetDebug(errMsg)
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
		return User{}, errRes
	}

	return resSlice[0], nil
}

func (s *UserSrvc) GetUsernames(ctx context.Context,
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
