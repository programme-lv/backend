package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

// ChangeUsername verifies the current password and updates the account name.
func (s *userSrvc) ChangeUsername(ctx context.Context, userUUID uuid.UUID, username, currentPassword string) (string, srvcerror.E) {
	log := ctxlog.FromContext(ctx)
	var currentUsername, hash string
	err := s.postgres.QueryRow(ctx, `SELECT username, bcrypt_pwd FROM users WHERE uuid = $1`, userUUID).Scan(&currentUsername, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		log.Error("read user for username change", "error", err)
		return "", srvcerror.InternalServerError()
	}
	if currentUsername == "admin" {
		return "", ErrUsernameChangeForbidden
	}
	if validationErr := validateUsername(username); validationErr != nil {
		return "", validationErr
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return "", ErrUsernameOrPasswordIncorrect
	}

	// Do not write using a password hash superseded by a concurrent password reset.
	var updated string
	err = s.postgres.QueryRow(ctx, `
		UPDATE users SET username = $1
		WHERE uuid = $2 AND bcrypt_pwd = $3 AND username <> 'admin'
		RETURNING username
	`, username, userUUID, hash).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUsernameOrPasswordIncorrect
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_username_key" {
			return "", ErrUsernameExists
		}
		log.Error("update username", "error", err)
		return "", srvcerror.InternalServerError()
	}
	return updated, nil
}
