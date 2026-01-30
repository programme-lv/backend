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

type userSrvc struct {
	postgres *pgxpool.Pool
}

type UserService interface {
	GetUserByUsername(ctx context.Context, username string) (User, srvcerror.E)
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (User, srvcerror.E)
	GetUsernames(ctx context.Context, uuids []uuid.UUID) ([]string, srvcerror.E)
	Login(ctx context.Context, username string, password string) (*User, srvcerror.E)
	CreateUser(ctx context.Context, user CreateUserParams) (*User, srvcerror.E)
}

func NewUserService(pg *pgxpool.Pool) *userSrvc {
	return &userSrvc{
		postgres: pg,
	}
}

func (s *userSrvc) GetUserByUsername(ctx context.Context, username string) (res User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get user by username")

	allUsers, selectAllUsersErr := selectAllUsers(s.postgres)
	if selectAllUsersErr != nil {
		l.Error("list users", "error", selectAllUsersErr)
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

func (s *userSrvc) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (res User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get user by uuid")

	allUsers, selectErr := selectAllUsers(s.postgres)
	if selectErr != nil {
		l.Error("list users", "error", selectErr)
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
		l.Warn("user with UUID not found", "uuid", uuid)
		return User{}, ErrUserNotFound
	}

	return resSlice[0], nil
}

func (s *userSrvc) GetUsernames(ctx context.Context,
	uuids []uuid.UUID) ([]string, srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get usernames")

	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		l.Error("list users", "error", err)
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
