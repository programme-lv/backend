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
	GetUserByUsername(ctx context.Context, username string) (User, srvcerror.E)
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

func (s *UserSrvc) GetUserByUsername(ctx context.Context, username string) (res User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get user by username")

	allUsers, selectAllUsersErr := selectAllUsers(s.postgres)
	if selectAllUsersErr != nil {
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
		return User{}, ErrUserNotFound
	}

	return resSlice[0], nil
}

func (s *UserSrvc) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (res User, err error) {
	l := ctxlog.FromContext(ctx).With("query", "get user by uuid")

	allUsers, selectErr := selectAllUsers(s.postgres)
	if selectErr != nil {
		l.Error("failed to list users", "error", selectErr)
		return User{}, srvcerror.InternalServerError()
	}

	var resSlice []User
	for _, user := range allUsers {
		if user.UUID == uuid {
			if len(resSlice) == 1 {
				l.Error("multiple users with the same UUID", "uuid", uuid)
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
		l.Error("user with UUID not found", "uuid", uuid)
		return User{}, ErrUserNotFound
	}

	return resSlice[0], nil
}

func (s *UserSrvc) GetUsernames(ctx context.Context,
	uuids []uuid.UUID) ([]string, error) {
	l := ctxlog.FromContext(ctx).With("query", "get usernames")

	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		l.Error("failed to list users", "error", err)
		return nil, srvcerror.InternalServerError()
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
			return nil, ErrUserNotFound
		}
	}

	return usernames, nil
}
