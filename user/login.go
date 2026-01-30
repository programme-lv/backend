package user

import (
	"context"

	"github.com/programme-lv/backend/common/ctxlog"
	"golang.org/x/crypto/bcrypt"
)

func (s *UserSrvc) Login(ctx context.Context, username string, password string) (res *User, err error) {
	l := ctxlog.FromContext(ctx).With("cmd", "login")

	allUsers, selectErr := selectAllUsers(s.postgres)
	if selectErr != nil {
		l.Error("failed to list users", "error", selectErr)
		return nil, newErrInternalSE()
	}

	for _, user := range allUsers {
		if user.Username == username {
			err = bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(password))
			if err == nil {
				return &User{
					UUID:      user.UUID,
					Username:  user.Username,
					Email:     user.Email,
					Firstname: &user.Firstname,
					Lastname:  &user.Lastname,
				}, nil
			}
		}
	}

	return nil, newErrUsernameOrPasswordIncorrect()
}
