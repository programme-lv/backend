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

	userTableName := os.Getenv("DDB_USER_TABLE_NAME")
	if userTableName == "" {
		log.Fatalf(ctx,
			errors.New("DDB_USER_TABLE_NAME is not set"),
			"cant read DDB_USER_TABLE_NAME from env in new user service constructor")
	}

	return &userssrvc{
		jwtKey: []byte(jwtKey),
		ddbUserTable: NewDynamoDbUsersTable(
			dynamodbClient, userTableName),
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

// GetUserByUsername implements users.Service.
func (s *userssrvc) GetUserByUsername(ctx context.Context, p *usergen.GetUserByUsernamePayload) (res *usergen.User, err error) {
	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	var resSlice []usergen.User = make([]usergen.User, 0)
	for _, user := range allUsers {
		if user.Username == p.Username {
			if len(resSlice) == 1 {
				return nil, usergen.InternalError("vairāki lietotāji ar vienādu lietotājvārdu")
			}

			firstname := ""
			if user.Firstname != nil {
				firstname = *user.Firstname
			}

			lastname := ""
			if user.Lastname != nil {
				lastname = *user.Lastname
			}

			genUser := usergen.User{
				UUID:      user.Uuid,
				Username:  user.Username,
				Email:     user.Email,
				Firstname: firstname,
				Lastname:  lastname,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		return nil, usergen.NotFound("lietotājs ar šādu lietotājvārdu neeksistē")
	}

	return &resSlice[0], nil
}
