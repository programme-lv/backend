package user

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func (s *UserSrvc) Login(ctx context.Context, username string, password string) (res *User, err error) {
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return nil, newErrInternalSE().SetDebug(errMsg)
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
