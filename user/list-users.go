package user

import (
	"context"

	usergen "github.com/programme-lv/backend/gen/users"
)

// ListUsers implements users.Service.
func (s *userssrvc) ListUsers(ctx context.Context) (res []*usergen.User, err error) {
	users, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, usergen.InternalError("failed to list users from ddb")
	}
	res = make([]*usergen.User, 0)
	for _, user := range users {
		firstname := ""
		if user.Firstname != nil {
			firstname = *user.Firstname
		}
		lastname := ""
		if user.Lastname != nil {
			lastname = *user.Lastname
		}

		res = append(res, &usergen.User{
			UUID:      user.Uuid,
			Username:  user.Username,
			Email:     user.Email,
			Firstname: firstname,
			Lastname:  lastname,
		})
	}
	return res, nil
}
