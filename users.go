package programmelv

import (
	"context"
	"fmt"

	users "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
	"goa.design/goa/v3/security"
)

// users service example implementation.
// The example methods log the requests and return zero values.
type userssrvc struct{}

// NewUsers returns the users service implementation.
func NewUsers() users.Service {
	return &userssrvc{}
}

// JWTAuth implements the authorization logic for service "users" for the "jwt"
// security scheme.
func (s *userssrvc) JWTAuth(ctx context.Context, token string, scheme *security.JWTScheme) (context.Context, error) {
	//
	// TBD: add authorization logic.
	//
	// In case of authorization failure this function should return
	// one of the generated error structs, e.g.:
	//
	//    return ctx, myservice.MakeUnauthorizedError("invalid token")
	//
	// Alternatively this function may return an instance of
	// goa.ServiceError with a Name field value that matches one of
	// the design error names, e.g:
	//
	//    return ctx, goa.PermanentError("unauthorized", "invalid token")
	//
	return ctx, fmt.Errorf("not implemented")
}

// List all users
func (s *userssrvc) ListUsers(ctx context.Context, p *users.SecurePayload) (res []*users.User, err error) {
	log.Printf(ctx, "users.listUsers")
	return
}

// Get a user by UUID
func (s *userssrvc) GetUser(ctx context.Context, p *users.SecureUUIDPayload) (res *users.User, err error) {
	res = &users.User{}
	log.Printf(ctx, "users.getUser")
	return
}

// Create a new user
func (s *userssrvc) CreateUser(ctx context.Context, p *users.UserPayload) (res *users.User, err error) {
	res = &users.User{}
	log.Printf(ctx, "users.createUser")
	return
}

// Update an existing user
func (s *userssrvc) UpdateUser(ctx context.Context, p *users.UpdateUserPayload) (res *users.User, err error) {
	res = &users.User{}
	log.Printf(ctx, "users.updateUser")
	return
}

// Delete a user
func (s *userssrvc) DeleteUser(ctx context.Context, p *users.SecureUUIDPayload) (err error) {
	log.Printf(ctx, "users.deleteUser")
	return
}

// User login
func (s *userssrvc) Login(ctx context.Context, p *users.LoginPayload) (res string, err error) {
	log.Printf(ctx, "users.login")
	return
}

// Query current JWT
func (s *userssrvc) QueryCurrentJWT(ctx context.Context, p *users.SecurePayload) (res string, err error) {
	log.Printf(ctx, "users.queryCurrentJWT")
	return
}
