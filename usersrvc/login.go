package usersrvc

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type LoginParams struct {
	Username string
	Password string
}

func (s *UserService) Login(ctx context.Context, p *LoginParams) (res *User, err error) {
	allUsers, err := selectAllUsers(s.postgres)
	if err != nil {
		errMsg := fmt.Errorf("error listing users: %w", err)
		return nil, newErrInternalSE().SetDebug(errMsg)
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			err = bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(p.Password))
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
