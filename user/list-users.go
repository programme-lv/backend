package user

import (
	"context"
)

// ListUsers implements users.Service.
func (s *UserService) ListUsers(ctx context.Context) (res []*User, err error) {
	users, err := s.ddbUserTable.List(ctx)
	if err != nil {
		// TODO: log "failed to list users from ddb"
		return nil, newErrInternalServerError()
	}
	res = make([]*User, 0)
	for _, user := range users {
		res = append(res, &User{
			UUID:      user.Uuid,
			Username:  user.Username,
			Email:     user.Email,
			Firstname: user.Firstname,
			Lastname:  user.Lastname,
		})
	}
	return res, nil
}
