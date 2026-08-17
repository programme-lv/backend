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

	tx, txErr := s.postgres.Begin(ctx)
	if txErr != nil {
		l.Error("begin change password tx", "error", txErr)
		return srvcerror.InternalServerError()
	}
	defer tx.Rollback(ctx)

	tag, updErr := tx.Exec(ctx, `
		UPDATE users SET bcrypt_pwd = $1, pwd_changed_at = $2 WHERE uuid = $3
	`, string(bcryptPwd), time.Now(), userUUID)
	if updErr != nil {
		l.Error("update password", "error", updErr)
		return srvcerror.InternalServerError()
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	if _, invErr := tx.Exec(ctx, `
		UPDATE user_email_tokens
		SET used_at = NOW()
		WHERE user_uuid = $1 AND purpose = $2 AND used_at IS NULL
	`, userUUID, purposePasswordReset); invErr != nil {
		l.Error("invalidate password reset tokens", "error", invErr)
		return srvcerror.InternalServerError()
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		l.Error("commit change password", "error", commitErr)
		return srvcerror.InternalServerError()
	}

	return nil
}
