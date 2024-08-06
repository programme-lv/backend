package user

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb" // assuming custom Latvian translations
	"github.com/programme-lv/backend/auth"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
	"goa.design/goa/v3/security"
	"golang.org/x/crypto/bcrypt"
)

type userssrvc struct {
	jwtKey       []byte
	ddbUserTable *DynamoDbUserTable
}

func NewUsers(ctx context.Context) usergen.Service {
	// read jwt key from env
	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatalf(ctx,
			errors.New("JWT_KEY is not set"),
			"cant read JWT_KEY from env in new user service constructor")
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

type ClaimsKey string

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
