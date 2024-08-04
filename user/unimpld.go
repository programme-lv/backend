package user

import (
	"context"

	usergen "github.com/programme-lv/backend/gen/users"
)

// DeleteUser implements users.Service.
func (s *userssrvc) DeleteUser(context.Context, *usergen.SecureUUIDPayload) (err error) {
	panic("unimplemented")
}

// GetUser implements users.Service.
func (s *userssrvc) GetUser(context.Context, *usergen.SecureUUIDPayload) (res *usergen.User, err error) {
	panic("unimplemented")
}

// ListUsers implements users.Service.
func (s *userssrvc) ListUsers(context.Context, *usergen.ListUsersPayload) (res []*usergen.User, err error) {
	panic("unimplemented")
}

// UpdateUser implements users.Service.
func (s *userssrvc) UpdateUser(context.Context, *usergen.UpdateUserPayload) (res *usergen.User, err error) {
	panic("unimplemented")
}
