package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/lv"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	lv_translations "github.com/go-playground/validator/v10/translations/lv" // assuming custom Latvian translations
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	"github.com/programme-lv/backend/auth"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
	"goa.design/goa/v3/security"
	"golang.org/x/crypto/bcrypt"
)

type userssrvc struct {
	jwtKey       []byte
	ddbUserTable *DynamoDbUserTable
	validate     *validator.Validate
	uni          *ut.UniversalTranslator
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

	// Initialize validator and translator
	validate := validator.New()

	// Set up translation for English and Latvian
	en := en.New()
	lv := lv.New()
	uni := ut.New(lv, lv, en)

	transEn, _ := uni.GetTranslator("en")
	transLv, _ := uni.GetTranslator("lv")

	en_translations.RegisterDefaultTranslations(validate, transEn)
	lv_translations.RegisterDefaultTranslations(validate, transLv)

	return &userssrvc{
		jwtKey: []byte(jwtKey),
		ddbUserTable: NewDynamoDbUsersTable(
			dynamodbClient, "ProglvUsers"),
		validate: validate,
		uni:      uni,
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

func mustToJson(x any) string {
	b, _ := json.Marshal(x)
	return string(b)
}

func (s *userssrvc) CreateUser(ctx context.Context, p *usergen.UserPayload) (res *usergen.User, err error) {
	uni := s.uni
	// en, _ := uni.GetTranslator("en")
	lv, _ := uni.GetTranslator("lv")
	lv.Add("Username", "lietotājvārds", true)
	input := struct {
		Username  string  `validate:"required,min=2,max=32"`
		Email     string  `validate:"required,email,max=320"`
		Firstname *string `validate:"omitempty,max=25"`
		Lastname  *string `validate:"omitempty,max=25"`
		Password  string  `validate:"required,min=8"`
	}{
		Username: p.Username,
		Email:    p.Email,
		Password: p.Password,
	}
	if p.Firstname != "" {
		input.Firstname = &p.Firstname
	}
	if p.Lastname != "" {
		input.Lastname = &p.Lastname
	}

	err = s.validate.Struct(input)
	if err != nil {
		validationErrs := err.(validator.ValidationErrors)
		return nil, usergen.InvalidUserDetails(mustToJson(validationErrs.Translate(lv)))
	}

	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			return nil, usergen.UsernameExists(fmt.Sprintf(
				"lietotājvārds %s jau eksistē / username %s already exists", p.Username, p.Username,
			))
		}

		if user.Email == p.Email {
			return nil, usergen.EmailExists(fmt.Sprintf(
				"epasts %s jau eksistē / email %s already exists", p.Email, p.Email,
			))
		}
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
