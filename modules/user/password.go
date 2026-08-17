package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

func (s *userSrvc) ChangePassword(ctx context.Context, userUUID uuid.UUID, current, newPassword string) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "change password")

	if validateErr := validatePassword(newPassword); validateErr != nil {
		return validateErr
	}

	user, selectErr := selectUserByUUID(ctx, s.postgres, userUUID)
	if selectErr != nil {
		if errors.Is(selectErr, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		l.Error("get user by uuid", "error", selectErr)
		return srvcerror.InternalServerError()
	}

	if bcrypt.CompareHashAndPassword([]byte(user.BcryptPwd), []byte(current)) != nil {
		return ErrUsernameOrPasswordIncorrect
	}

	bcryptPwd, bcryptErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if bcryptErr != nil {
		l.Error("hash new password", "error", bcryptErr)
		return srvcerror.InternalServerError()
	}

	tag, updErr := s.postgres.Exec(ctx, `
		UPDATE users SET bcrypt_pwd = $1, pwd_changed_at = $2 WHERE uuid = $3
	`, string(bcryptPwd), time.Now(), userUUID)
	if updErr != nil {
		l.Error("update password", "error", updErr)
		return srvcerror.InternalServerError()
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	return nil
}
