package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

func (s *userSrvc) Login(ctx context.Context, login string, password string) (res *User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("cmd", "login")

	candidates, selectErr := selectLoginCandidates(ctx, s.postgres, login)
	if selectErr != nil {
		l.Error("get login candidates", "error", selectErr)
		return nil, srvcerror.InternalServerError()
	}

	var matched *dbUser
	for i := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(candidates[i].BcryptPwd), []byte(password)) != nil {
			continue
		}
		if matched != nil {
			return nil, ErrUsernameOrPasswordIncorrect
		}
		matched = &candidates[i]
	}
	if matched == nil {
		return nil, ErrUsernameOrPasswordIncorrect
	}

	return &User{
		UUID:          matched.UUID,
		Username:      matched.Username,
		Email:         matched.Email,
		Firstname:     &matched.Firstname,
		Lastname:      &matched.Lastname,
		EmailVerified: matched.EmailVerified,
	}, nil
}

// selectLoginCandidates matches the identifier as an exact username or a
// case-insensitive email. New usernames cannot contain @, but legacy ones may,
// and email uniqueness is still case-sensitive, so one identifier can match
// more than one account. Login verifies the password against every distinct
// candidate and succeeds only when exactly one matches.
func selectLoginCandidates(ctx context.Context, pg *pgxpool.Pool, login string) ([]dbUser, error) {
	rows, err := pg.Query(ctx, `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
		FROM users
		WHERE username = $1 OR LOWER(email) = LOWER($1)
		ORDER BY uuid
	`, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]dbUser, 0, 2)
	seen := make(map[uuid.UUID]struct{}, 2)
	for rows.Next() {
		var user dbUser
		if err := rows.Scan(
			&user.UUID,
			&user.Firstname,
			&user.Lastname,
			&user.Username,
			&user.Email,
			&user.BcryptPwd,
			&user.CreatedAt,
			&user.EmailVerified,
		); err != nil {
			return nil, err
		}
		if _, ok := seen[user.UUID]; ok {
			continue
		}
		seen[user.UUID] = struct{}{}
		candidates = append(candidates, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
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
