package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	"github.com/programme-lv/backend/auth"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
	"goa.design/goa/v3/security"
	"golang.org/x/crypto/bcrypt"
)

// users service example implementation.
// The example methods log the requests and return zero values.
type userssrvc struct {
	jwtKey       []byte
	ddbUserTable *DynamoDbUserTable
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

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
		config.WithLogger(log.AsAWSLogger(ctx)))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	return &userssrvc{
		jwtKey: []byte(jwtKey),
		ddbUserTable: NewDynamoDbUsersTable(
			dynamodbClient, "ProglvUsers"),
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
		fmt.Println(err)
		return ctx, ErrInvalidToken
	}

	scopesInToken := claims.Scopes

	if err := scheme.Validate(scopesInToken); err != nil {
		fmt.Println("invalid scopes in token")
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
	const maxFirstnameLastnameLength = len("pretpulkstenraditajvirziens")
	const maxEmailLength = 320
	const maxUsernameLength = 32
	const minPasswordLength = 8
	const minUsernameLength = 2

	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, err
	}

	if len(p.Username) < minUsernameLength {
		return nil, usergen.InvalidUserDetails("lietotājvārds ir pārāk īss")
	}

	if len(p.Username) > maxUsernameLength {
		return nil, usergen.InvalidUserDetails("lietotājvārds ir pārāk garšs")
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			return nil, usergen.UsernameExists(fmt.Sprintf(
				"lietotājvārds %s jau eksistē", p.Username,
			))
		}
	}

	if !validEmail(p.Email) {
		return nil, usergen.InvalidUserDetails("nekorekts e-pasts")
	}

	if len(p.Email) > maxEmailLength {
		return nil, usergen.InvalidUserDetails("epasts ir pārāk garšs")
	}

	for _, user := range allUsers {
		if user.Email == p.Email {
			return nil, usergen.EmailExists(fmt.Sprintf(
				"epasts %s jau eksistē", p.Email,
			))
		}
	}

	if len(p.Firstname) > maxFirstnameLastnameLength {
		return nil, usergen.InvalidUserDetails("vārds ir pārāk garšs")
	}

	if len(p.Lastname) > maxFirstnameLastnameLength {
		return nil, usergen.InvalidUserDetails("uzvārds ir pārāk garšs")
	}

	if len(p.Password) < minPasswordLength {
		return nil, usergen.InvalidUserDetails(fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minPasswordLength))
	}

	bcryptPwd, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("error hashing password")
	}

	uuid := uuid.New()

	var firstname *string = nil
	if p.Firstname != "" {
		firstname = &p.Firstname
	}

	var lastname *string = nil
	if p.Lastname != "" {
		lastname = &p.Lastname
	}

	row := &UserRow{
		Uuid:      uuid.String(),
		Username:  p.Username,
		Email:     p.Email,
		BcryptPwd: bcryptPwd,
		Firstname: firstname,
		Lastname:  lastname,
		Version:   0,
		CreatedAt: time.Now(),
	}

	err = s.ddbUserTable.Save(ctx, row)
	if err != nil {
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, usergen.InternalError("version conflict saving user")
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, usergen.InternalError("error saving user")
		}
	}

	res = &usergen.User{
		UUID:      uuid.String(),
		Username:  p.Username,
		Email:     p.Email,
		Firstname: p.Firstname,
		Lastname:  p.Lastname,
	}

	return res, nil
}

// Update an existing user
func (s *userssrvc) UpdateUser(ctx context.Context, p *usergen.UpdateUserPayload) (res *usergen.User, err error) {
	res = &usergen.User{}
	log.Printf(ctx, "users.updateUser")
	return
}

// Delete a user
func (s *userssrvc) DeleteUser(ctx context.Context, p *usergen.SecureUUIDPayload) (err error) {
	log.Printf(ctx, "users.deleteUser")
	return
}

// User login
func (s *userssrvc) Login(ctx context.Context, p *usergen.LoginPayload) (res string, err error) {
	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return "", fmt.Errorf("error listing users: %w", err)
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			err = bcrypt.CompareHashAndPassword(user.BcryptPwd, []byte(p.Password))
			if err == nil {
				token, err := auth.GenerateJWT(
					user.Username,
					user.Email, user.Uuid,
					user.Firstname, user.Lastname,
					s.jwtKey)
				if err != nil {
					return "", fmt.Errorf("error generating JWT: %w", err)
				}
				if token == "" {
					return "", fmt.Errorf("error generating JWT")
				}
				return token, nil
			}
		}
	}

	return "", usergen.InvalidCredentials("lietotājvārds vai parole nav pareiza")
}

// // Query current JWT
// func (s *userssrvc) QueryCurrentJWT(ctx context.Context, p *usergen.QueryCurrentJWTPayload) (res string, err error) {
// 	// claims := ctx.Value(ClaimsKey("claims")).(auth.Claims)
// }

// QueryCurrentJWT implements users.Service.
func (s *userssrvc) QueryCurrentJWT(ctx context.Context, p *usergen.QueryCurrentJWTPayload) (res *usergen.JWTClaims, err error) {
	claims := ctx.Value(ClaimsKey("claims")).(*auth.Claims)

	var expiresAt *string = nil
	if claims.ExpiresAt != nil {
		expiresAt = new(string)
		*expiresAt = claims.ExpiresAt.String()
	}

	var issuedAt *string = nil
	if claims.IssuedAt != nil {
		issuedAt = new(string)
		*issuedAt = claims.IssuedAt.String()
	}

	var notBefore *string = nil
	if claims.NotBefore != nil {
		notBefore = new(string)
		*notBefore = claims.NotBefore.String()
	}

	res = &usergen.JWTClaims{
		Username:  &claims.Username,
		Firstname: claims.Firstname,
		Lastname:  claims.Lastname,
		Email:     &claims.Email,
		UUID:      &claims.UUID,
		Scopes:    claims.Scopes,
		Issuer:    &claims.Issuer,
		Subject:   &claims.Subject,
		Audience:  claims.Audience,
		ExpiresAt: expiresAt,
		IssuedAt:  issuedAt,
		NotBefore: notBefore,
	}

	return res, nil
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
