package user

import (
	"context"
	"fmt"

	// assuming custom Latvian translations
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

type UserSrvc struct {
	postgres *pgxpool.Pool
}

type UserSrvcClient interface {
	GetUserByUsername(ctx context.Context, username string) (User, *srvcerror.Error)
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (User, error)
	GetUsernames(ctx context.Context, uuids []uuid.UUID) ([]string, error)
	Login(ctx context.Context, username string, password string) (*User, error)
	CreateUser(ctx context.Context, user CreateUserParams) (*User, error)
}

func NewUserService(pg *pgxpool.Pool) UserSrvcClient {
	return &UserSrvc{
		postgres: pg,
	}
}

func (s *UserSrvc) GetUserByUsername(ctx context.Context, username string) (res User, err *srvcerror.Error) {
	l := ctxlog.FromContext(ctx)
	var selectAllUsersErr error
	var allUsers []dbUser
	allUsers, selectAllUsersErr = selectAllUsers(s.postgres)
	if err != nil {
		// errMsg := fmt.Errorf("error listing users: %w", err)
		l.Error("failed to list users", "error", selectAllUsersErr)
		return User{}, srvcerror.InternalServerError()
	}

	var resSlice []User = make([]User, 0)
	for _, user := range allUsers {
		if user.Username == username {
			if len(resSlice) == 1 {
				format := "multiple users with the same username: %s"
				errMsg := fmt.Errorf(format, username)
				l.Error("multiple users with the same username", "username", username, "error", errMsg)
				return User{}, srvcerror.InternalServerError()
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
		l.Error("user with username not found", "username", username, "error", errMsg)
		return User{}, srvcerror.InternalServerError()
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
