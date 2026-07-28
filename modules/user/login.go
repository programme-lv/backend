package user

import (
	"context"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

func (s *userSrvc) Login(ctx context.Context, username string, password string) (res *User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("cmd", "login")

	allUsers, selectErr := selectAllUsers(s.postgres)
	if selectErr != nil {
		l.Error("list users", "error", selectErr)
		return nil, newErrInternalSE()
	}

	for _, user := range allUsers {
		if user.Username == username {
			bcryptErr := bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(password))
			if bcryptErr == nil {
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
