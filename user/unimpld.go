package user

import (
	"context"

	usergen "github.com/programme-lv/backend/gen/users"
)

// DeleteUser implements users.Service.
func (s *UsersSrvc) DeleteUser(context.Context, *usergen.SecureUUIDPayload) (err error) {
	panic("unimplemented")
}

// GetUser implements users.Service.
func (s *UsersSrvc) GetUser(context.Context, *usergen.SecureUUIDPayload) (res *usergen.User, err error) {
	panic("unimplemented")
}
