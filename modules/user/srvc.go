package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/user/mail"
)

type EmailFlowConfig struct {
	WebsiteBaseURL  string
	ResetTokenTTL   time.Duration
	VerifyTokenTTL  time.Duration
	PerUserCooldown time.Duration
}

type userSrvc struct {
	postgres *pgxpool.Pool
	mailer   mail.Mailer
	emailCfg EmailFlowConfig
}

type UserService interface {
	GetUserByUsername(ctx context.Context, username string) (User, srvcerror.E)
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (User, srvcerror.E)
	Login(ctx context.Context, username string, password string) (*User, srvcerror.E)
	CreateUser(ctx context.Context, user CreateUserParams) (*User, srvcerror.E)
	RequestPasswordReset(ctx context.Context, login string) srvcerror.E
	ConfirmPasswordReset(ctx context.Context, token string, newPassword string) srvcerror.E
	RequestEmailVerification(ctx context.Context, userUUID uuid.UUID) srvcerror.E
	ConfirmEmailVerification(ctx context.Context, token string) srvcerror.E
	PasswordChangedAt(ctx context.Context, userUUID uuid.UUID) (time.Time, error)
}

func NewUserService(pg *pgxpool.Pool, mailer mail.Mailer, emailCfg EmailFlowConfig) *userSrvc {
	if mailer == nil {
		mailer = mail.NewNoopMailer()
	}
	if emailCfg.ResetTokenTTL <= 0 {
		emailCfg.ResetTokenTTL = time.Hour
	}
	if emailCfg.VerifyTokenTTL <= 0 {
		emailCfg.VerifyTokenTTL = 24 * time.Hour
	}
	if emailCfg.PerUserCooldown <= 0 {
		emailCfg.PerUserCooldown = 5 * time.Minute
	}
	return &userSrvc{
		postgres: pg,
		mailer:   mailer,
		emailCfg: emailCfg,
	}
}

func (s *userSrvc) GetUserByUsername(ctx context.Context, username string) (res User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get user by username")

	allUsers, selectAllUsersErr := selectAllUsers(s.postgres)
	if selectAllUsersErr != nil {
		l.Error("list users", "error", selectAllUsersErr)
		return User{}, srvcerror.InternalServerError()
	}

	var resSlice []User = make([]User, 0)
	for _, user := range allUsers {
		if user.Username == username {
			if len(resSlice) == 1 {
				format := "multiple users with the same username: %s"
				errMsg := fmt.Errorf(format, username)
				l.Error("multiple users with the same username", "username", username, "error", errMsg)
				return User{}, srvcerror.InternalServerError()
			}

			genUser := User{
				UUID:          user.UUID,
				Username:      user.Username,
				Email:         user.Email,
				Firstname:     &user.Firstname,
				Lastname:      &user.Lastname,
				EmailVerified: user.EmailVerified,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		return User{}, ErrUserNotFound
	}

	return resSlice[0], nil
}

func (s *userSrvc) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (res User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("query", "get user by uuid")

	allUsers, selectErr := selectAllUsers(s.postgres)
	if selectErr != nil {
		l.Error("list users", "error", selectErr)
		return User{}, srvcerror.InternalServerError()
	}

	var resSlice []User
	for _, user := range allUsers {
		if user.UUID == uuid {
			if len(resSlice) == 1 {
				l.Error("multiple users with the same UUID", "uuid", uuid)
				return User{}, srvcerror.InternalServerError()
			}

			genUser := User{
				UUID:          user.UUID,
				Username:      user.Username,
				Email:         user.Email,
				Firstname:     &user.Firstname,
				Lastname:      &user.Lastname,
				EmailVerified: user.EmailVerified,
			}
			resSlice = append(resSlice, genUser)
		}
	}
	if len(resSlice) == 0 {
		return User{}, ErrUserNotFound
	}

	return resSlice[0], nil
}

func (s *userSrvc) PasswordChangedAt(ctx context.Context, userUUID uuid.UUID) (time.Time, error) {
	var changedAt time.Time
	err := s.postgres.QueryRow(ctx, `
		SELECT pwd_changed_at FROM users WHERE uuid = $1
	`, userUUID).Scan(&changedAt)
	return changedAt, err
}
