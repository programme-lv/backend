package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/user/mail"
	"golang.org/x/crypto/bcrypt"
)

const (
	purposePasswordReset = "password_reset"
	purposeEmailVerify   = "email_verify"
)

func (s *userSrvc) RequestPasswordReset(ctx context.Context, login string) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "request password reset")

	login = trimLogin(login)
	if login == "" {
		return nil
	}

	user, found, err := findUserByUsernameOrEmail(ctx, s.postgres, login)
	if err != nil {
		l.Error("resolve user for password reset", "error", err)
		return nil
	}
	if !found {
		return nil
	}

	if cooled, coolErr := s.isWithinCooldown(ctx, user.UUID, purposePasswordReset); coolErr != nil {
		l.Error("check password reset cooldown", "error", coolErr)
		return nil
	} else if cooled {
		return nil
	}

	if s.emailCfg.WebsiteBaseURL == "" {
		l.Error("skip password reset email: WEBSITE_PUBLIC_BASE_URL is empty")
		return nil
	}

	rawToken, tokenHash, genErr := generateEmailToken()
	if genErr != nil {
		l.Error("generate password reset token", "error", genErr)
		return nil
	}

	actionURL := s.websiteURL("/reset-password", rawToken)
	rendered, renderErr := mail.RenderPasswordReset(mail.TemplateData{
		Username:   user.Username,
		ActionURL:  actionURL,
		ExpiryNote: fmt.Sprintf("Saite derīga %s.", formatTTL(s.emailCfg.ResetTokenTTL)),
	})
	if renderErr != nil {
		l.Error("render password reset email", "error", renderErr)
		return nil
	}

	if sendErr := s.mailer.Send(ctx, mail.Message{
		To:       user.Email,
		Subject:  rendered.Subject,
		TextBody: rendered.TextBody,
		HTMLBody: rendered.HTMLBody,
	}); sendErr != nil {
		l.Error("send password reset email", "error", sendErr)
		return nil
	}

	expiresAt := time.Now().Add(s.emailCfg.ResetTokenTTL)
	if insertErr := insertEmailToken(ctx, s.postgres, user.UUID, purposePasswordReset, tokenHash, expiresAt); insertErr != nil {
		l.Error("store password reset token", "error", insertErr)
	}

	return nil
}

func (s *userSrvc) ConfirmPasswordReset(ctx context.Context, token string, newPassword string) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "confirm password reset")

	if validateErr := validatePassword(newPassword); validateErr != nil {
		return validateErr
	}

	tokenHash := hashEmailToken(token)

	bcryptPwd, bcryptErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if bcryptErr != nil {
		l.Error("hash new password", "error", bcryptErr)
		return newErrInternalSE()
	}

	tx, txErr := s.postgres.Begin(ctx)
	if txErr != nil {
		l.Error("begin password reset tx", "error", txErr)
		return newErrInternalSE()
	}
	defer tx.Rollback(ctx)

	row, err := loadValidEmailTokenForUpdate(ctx, tx, purposePasswordReset, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return newErrEmailTokenInvalid()
		}
		l.Error("load password reset token", "error", err)
		return newErrInternalSE()
	}

	tag, markErr := tx.Exec(ctx, `
		UPDATE user_email_tokens SET used_at = NOW() WHERE uuid = $1 AND used_at IS NULL
	`, row.UUID)
	if markErr != nil {
		l.Error("mark password reset token used", "error", markErr)
		return newErrInternalSE()
	}
	if tag.RowsAffected() != 1 {
		return newErrEmailTokenInvalid()
	}

	if _, updErr := tx.Exec(ctx, `
		UPDATE users SET bcrypt_pwd = $1, pwd_changed_at = NOW() WHERE uuid = $2
	`, string(bcryptPwd), row.UserUUID); updErr != nil {
		l.Error("update password", "error", updErr)
		return newErrInternalSE()
	}

	if _, invErr := tx.Exec(ctx, `
		UPDATE user_email_tokens
		SET used_at = NOW()
		WHERE user_uuid = $1 AND purpose = $2 AND used_at IS NULL AND uuid <> $3
	`, row.UserUUID, purposePasswordReset, row.UUID); invErr != nil {
		l.Error("invalidate other password reset tokens", "error", invErr)
		return newErrInternalSE()
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		l.Error("commit password reset", "error", commitErr)
		return newErrInternalSE()
	}

	return nil
}

func (s *userSrvc) RequestEmailVerification(ctx context.Context, userUUID uuid.UUID) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "request email verification")

	user, err := selectUserByUUID(ctx, s.postgres, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		l.Error("load user for email verification", "error", err)
		return newErrInternalSE()
	}
	if user.EmailVerified {
		return nil
	}

	if cooled, coolErr := s.isWithinCooldown(ctx, user.UUID, purposeEmailVerify); coolErr != nil {
		l.Error("check email verify cooldown", "error", coolErr)
		return newErrInternalSE()
	} else if cooled {
		return newErrEmailSendTooFrequent()
	}

	if sendErr := s.sendEmailVerification(ctx, user); sendErr != nil {
		l.Error("send email verification", "error", sendErr)
		return newErrInternalSE()
	}
	return nil
}

func (s *userSrvc) ConfirmEmailVerification(ctx context.Context, token string) srvcerror.E {
	l := ctxlog.FromContext(ctx).With("cmd", "confirm email verification")

	tokenHash := hashEmailToken(token)

	tx, txErr := s.postgres.Begin(ctx)
	if txErr != nil {
		l.Error("begin email verify tx", "error", txErr)
		return newErrInternalSE()
	}
	defer tx.Rollback(ctx)

	row, err := loadValidEmailTokenForUpdate(ctx, tx, purposeEmailVerify, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return newErrEmailTokenInvalid()
		}
		l.Error("load email verify token", "error", err)
		return newErrInternalSE()
	}

	tag, markErr := tx.Exec(ctx, `
		UPDATE user_email_tokens SET used_at = NOW() WHERE uuid = $1 AND used_at IS NULL
	`, row.UUID)
	if markErr != nil {
		l.Error("mark email verify token used", "error", markErr)
		return newErrInternalSE()
	}
	if tag.RowsAffected() != 1 {
		return newErrEmailTokenInvalid()
	}

	if _, updErr := tx.Exec(ctx, `
		UPDATE users SET email_verified = true WHERE uuid = $1
	`, row.UserUUID); updErr != nil {
		l.Error("mark email verified", "error", updErr)
		return newErrInternalSE()
	}

	if _, invErr := tx.Exec(ctx, `
		UPDATE user_email_tokens
		SET used_at = NOW()
		WHERE user_uuid = $1 AND purpose = $2 AND used_at IS NULL AND uuid <> $3
	`, row.UserUUID, purposeEmailVerify, row.UUID); invErr != nil {
		l.Error("invalidate other email verify tokens", "error", invErr)
		return newErrInternalSE()
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		l.Error("commit email verify", "error", commitErr)
		return newErrInternalSE()
	}

	return nil
}

func (s *userSrvc) sendEmailVerification(ctx context.Context, user dbUser) error {
	l := ctxlog.FromContext(ctx).With("cmd", "send email verification")

	if s.emailCfg.WebsiteBaseURL == "" {
		l.Info("skip email verification: WEBSITE_PUBLIC_BASE_URL is empty")
		return nil
	}

	rawToken, tokenHash, genErr := generateEmailToken()
	if genErr != nil {
		return genErr
	}

	actionURL := s.websiteURL("/verify-email", rawToken)
	rendered, renderErr := mail.RenderEmailVerify(mail.TemplateData{
		Username:   user.Username,
		ActionURL:  actionURL,
		ExpiryNote: fmt.Sprintf("Saite derīga %s.", formatTTL(s.emailCfg.VerifyTokenTTL)),
	})
	if renderErr != nil {
		return renderErr
	}

	if sendErr := s.mailer.Send(ctx, mail.Message{
		To:       user.Email,
		Subject:  rendered.Subject,
		TextBody: rendered.TextBody,
		HTMLBody: rendered.HTMLBody,
	}); sendErr != nil {
		l.Error("smtp send email verification", "error", sendErr)
		return sendErr
	}

	expiresAt := time.Now().Add(s.emailCfg.VerifyTokenTTL)
	if insertErr := insertEmailToken(ctx, s.postgres, user.UUID, purposeEmailVerify, tokenHash, expiresAt); insertErr != nil {
		return insertErr
	}
	return nil
}

func (s *userSrvc) websiteURL(path, token string) string {
	u, err := url.Parse(s.emailCfg.WebsiteBaseURL + path)
	if err != nil {
		return s.emailCfg.WebsiteBaseURL + path + "?token=" + url.QueryEscape(token)
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *userSrvc) isWithinCooldown(ctx context.Context, userUUID uuid.UUID, purpose string) (bool, error) {
	var createdAt time.Time
	err := s.postgres.QueryRow(ctx, `
		SELECT created_at
		FROM user_email_tokens
		WHERE user_uuid = $1 AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userUUID, purpose).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return time.Since(createdAt) < s.emailCfg.PerUserCooldown, nil
}

type emailTokenRow struct {
	UUID     uuid.UUID
	UserUUID uuid.UUID
}

type tokenQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func loadValidEmailTokenForUpdate(ctx context.Context, q tokenQuerier, purpose, tokenHash string) (emailTokenRow, error) {
	var row emailTokenRow
	err := q.QueryRow(ctx, `
		SELECT uuid, user_uuid
		FROM user_email_tokens
		WHERE token_hash = $1
		  AND purpose = $2
		  AND used_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE
	`, tokenHash, purpose).Scan(&row.UUID, &row.UserUUID)
	return row, err
}

func insertEmailToken(
	ctx context.Context,
	pg *pgxpool.Pool,
	userUUID uuid.UUID,
	purpose string,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := pg.Exec(ctx, `
		INSERT INTO user_email_tokens (uuid, user_uuid, purpose, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), userUUID, purpose, tokenHash, expiresAt)
	return err
}

func generateEmailToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashEmailToken(raw), nil
}

func hashEmailToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func findUserByUsernameOrEmail(ctx context.Context, pg *pgxpool.Pool, login string) (dbUser, bool, error) {
	user, err := selectUserByUsername(ctx, pg, login)
	if err == nil {
		return user, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return dbUser{}, false, err
	}

	user, err = selectUserByEmail(ctx, pg, login)
	if err == nil {
		return user, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return dbUser{}, false, nil
	}
	return dbUser{}, false, err
}

func selectUserByEmail(ctx context.Context, pg *pgxpool.Pool, email string) (dbUser, error) {
	var user dbUser
	err := pg.QueryRow(ctx, `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
		FROM users
		WHERE email = $1
	`, email).Scan(
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

func selectUserByUUID(ctx context.Context, pg *pgxpool.Pool, id uuid.UUID) (dbUser, error) {
	var user dbUser
	err := pg.QueryRow(ctx, `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
		FROM users
		WHERE uuid = $1
	`, id).Scan(
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

func trimLogin(login string) string {
	return strings.TrimSpace(login)
}

func formatTTL(d time.Duration) string {
	if d%time.Hour == 0 {
		h := int(d / time.Hour)
		if h == 1 {
			return "1 stundu"
		}
		return fmt.Sprintf("%d stundas", h)
	}
	if d%time.Minute == 0 {
		m := int(d / time.Minute)
		if m == 1 {
			return "1 minūti"
		}
		return fmt.Sprintf("%d minūtes", m)
	}
	return d.String()
}
