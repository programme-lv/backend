package users

import (
	"context"
	"errors"
	"os"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/auth"
	"github.com/programme-lv/backend/gen/users"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
	"goa.design/goa/v3/security"
)

// users service example implementation.
// The example methods log the requests and return zero values.
type userssrvc struct {
	jwtKey []byte
}

// NewUsers returns the users service implementation.
func NewUsers(ctx context.Context) usergen.Service {
	// read jwt key from env
	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatalf(ctx,
			errors.New("JWT_KEY is not set"),
			"cant read JWT_KEY from env in new user service contructor")
	}
	return &userssrvc{
		jwtKey: []byte(jwtKey),
	}
}

var (
	ErrInvalidToken       = usergen.Unauthorized("invalid token")
	ErrInvalidTokenScopes = usergen.Unauthorized("invalid scopes in token")
	ErrMissingScope       = usergen.Unauthorized("missing scope in token")
)

type ClaimsKey string

// JWTAuth implements the authorization logic for service "users" for the "jwt"
// security scheme.
func (s *userssrvc) JWTAuth(ctx context.Context, token string, scheme *security.JWTScheme) (context.Context, error) {
	claims, err := auth.ValidateJWT(token, s.jwtKey)
	if err != nil {
		return ctx, ErrInvalidToken
	}

	scopesInToken := claims.Scopes

	if err := scheme.Validate(scopesInToken); err != nil {
		return ctx, ErrMissingScope
	}

	ctx = context.WithValue(ctx, ClaimsKey("claims"), claims)
	return ctx, nil
}

// List all users
func (s *userssrvc) ListUsers(ctx context.Context, p *usergen.ListUsersPayload) (res []*usergen.User, err error) {
	log.Printf(ctx, "users.listUsers")
	return
}

// Get a user by UUID
func (s *userssrvc) GetUser(ctx context.Context, p *usergen.SecureUUIDPayload) (res *usergen.User, err error) {
	res = &usergen.User{}
	log.Printf(ctx, "users.getUser")
	return
}

// Create a new user
func (s *userssrvc) CreateUser(ctx context.Context, p *usergen.UserPayload) (res *usergen.User, err error) {
	uuid := uuid.New()

	row := &UserRow{
		Uuid:      uuid.New().String(),
		Username:  p.Username,
		Email:     p.Email,
		BcryptPwd: "",
		Firstname: &p.Firstname,
		Lastname:  &p.Lastname,
		Version:   0,
	}
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
	// auth.GenerateJWT()
	log.Printf(ctx, "payload: %+v", p)
	return
}

// Query current JWT
func (s *userssrvc) QueryCurrentJWT(ctx context.Context, p *users.QueryCurrentJWTPayload) (res string, err error) {
	log.Printf(ctx, "users.queryCurrentJWT")
	return
}
