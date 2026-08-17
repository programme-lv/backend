package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

func (s *userSrvc) Login(ctx context.Context, username string, password string) (res *User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("cmd", "login")

	user, selectErr := selectUserByUsername(ctx, s.postgres, username)
	if selectErr != nil {
		if errors.Is(selectErr, pgx.ErrNoRows) {
			return nil, ErrUsernameOrPasswordIncorrect
		}
		l.Error("get user by username", "error", selectErr)
		return nil, srvcerror.InternalServerError()
	}

	if bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(password)) != nil {
		return nil, ErrUsernameOrPasswordIncorrect
	}

	return &User{
		UUID:          user.UUID,
		Username:      user.Username,
		Email:         user.Email,
		Firstname:     &user.Firstname,
		Lastname:      &user.Lastname,
		EmailVerified: user.EmailVerified,
	}, nil
}

func selectUserByUsername(ctx context.Context, pg *pgxpool.Pool, username string) (dbUser, error) {
	var user dbUser
	err := pg.QueryRow(ctx, `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.UUID,
		&user.Firstname,
		&user.Lastname,
		&user.Username,
		&user.Email,
		&user.BcryptPwd,
		&user.CreatedAt,
		&user.EmailVerified,
	)
	return user, err
}
