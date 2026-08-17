package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

func (s *userSrvc) UpdateProfile(ctx context.Context, userUUID uuid.UUID, firstname, lastname string) (*User, srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("cmd", "update profile")

	if validateErr := validateFirstname(firstname); validateErr != nil {
		return nil, validateErr
	}
	if validateErr := validateLastname(lastname); validateErr != nil {
		return nil, validateErr
	}

	var row dbUser
	scanErr := s.postgres.QueryRow(ctx, `
		UPDATE users SET firstname = $1, lastname = $2 WHERE uuid = $3
		RETURNING uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
	`, firstname, lastname, userUUID).Scan(
		&row.UUID,
		&row.Firstname,
		&row.Lastname,
		&row.Username,
		&row.Email,
		&row.BcryptPwd,
		&row.CreatedAt,
		&row.EmailVerified,
	)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		l.Error("update profile", "error", scanErr)
		return nil, srvcerror.InternalServerError()
	}

	return &User{
		UUID:          row.UUID,
		Username:      row.Username,
		Email:         row.Email,
		Firstname:     &row.Firstname,
		Lastname:      &row.Lastname,
		EmailVerified: row.EmailVerified,
	}, nil
}
