package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserParams struct {
	Username  string
	Email     string
	Firstname *string
	Lastname  *string
	Password  string
}

func (s *userSrvc) CreateUser(ctx context.Context, p CreateUserParams) (res *User, err srvcerror.E) {
	l := ctxlog.FromContext(ctx).With("cmd", "create user")

	// Validate all fields
	if validateErr := validateUsername(p.Username); validateErr != nil {
		return nil, validateErr
	}
	if validateErr := validateEmail(p.Email); validateErr != nil {
		return nil, validateErr
	}
	if validateErr := validatePassword(p.Password); validateErr != nil {
		return nil, validateErr
	}
	if p.Firstname != nil {
		if validateErr := validateFirstname(*p.Firstname); validateErr != nil {
			return nil, validateErr
		}
	}
	if p.Lastname != nil {
		if validateErr := validateLastname(*p.Lastname); validateErr != nil {
			return nil, validateErr
		}
	}

	usernameExists, emailExists, selectErr := checkUserConflicts(ctx, s.postgres, p.Username, p.Email)
	if selectErr != nil {
		l.Error("check user conflicts", "error", selectErr)
		return nil, newErrInternalSE()
	}
	if usernameExists {
		return nil, newErrUsernameExists()
	}
	if emailExists {
		return nil, newErrEmailExists()
	}

	bcryptPwd, bcryptErr := bcrypt.GenerateFromPassword(
		[]byte(p.Password), bcrypt.DefaultCost)
	if bcryptErr != nil {
		l.Error("generate bcrypt password", "error", bcryptErr)
		return nil, newErrInternalSE()
	}

	firstname := ""
	if p.Firstname != nil {
		firstname = *p.Firstname
	}

	lastname := ""
	if p.Lastname != nil {
		lastname = *p.Lastname
	}

	row := &dbUser{
		UUID:          uuid.New(),
		Firstname:     firstname,
		Lastname:      lastname,
		Username:      p.Username,
		Email:         p.Email,
		BcryptPwd:     string(bcryptPwd),
		CreatedAt:     time.Now(),
		EmailVerified: false,
	}

	insertErr := insertUser(s.postgres, row)
	if insertErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(insertErr, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return nil, newErrUsernameExists()
			case "users_email_key":
				return nil, newErrEmailExists()
			}
		}
		l.Error("insert user", "error", insertErr)
		return nil, newErrInternalSE()
	}

	if sendErr := s.sendEmailVerification(ctx, *row); sendErr != nil {
		l.Error("send registration verification email", "error", sendErr)
	}

	res = &User{
		UUID:          row.UUID,
		Username:      row.Username,
		Email:         row.Email,
		Firstname:     &row.Firstname,
		Lastname:      &row.Lastname,
		EmailVerified: row.EmailVerified,
	}

	return res, nil
}

func checkUserConflicts(
	ctx context.Context,
	pg *pgxpool.Pool,
	username string,
	email string,
) (usernameExists, emailExists bool, err error) {
	err = pg.QueryRow(ctx, `
		SELECT
			EXISTS(SELECT 1 FROM users WHERE username = $1),
			EXISTS(SELECT 1 FROM users WHERE email = $2)
	`, username, email).Scan(&usernameExists, &emailExists)
	return usernameExists, emailExists, err
}

type dbUser struct {
	UUID          uuid.UUID
	Firstname     string
	Lastname      string
	Username      string
	Email         string
	BcryptPwd     string
	CreatedAt     time.Time
	EmailVerified bool
}

func selectAllUsers(pg *pgxpool.Pool) ([]dbUser, error) {
	rows, err := pg.Query(context.Background(), `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified
		FROM users
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []dbUser
	for rows.Next() {
		var user dbUser
		err := rows.Scan(
			&user.UUID,
			&user.Firstname,
			&user.Lastname,
			&user.Username,
			&user.Email,
			&user.BcryptPwd,
			&user.CreatedAt,
			&user.EmailVerified,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func insertUser(pg *pgxpool.Pool, user *dbUser) error {
	_, err := pg.Exec(context.Background(), `
		INSERT INTO users (uuid, firstname, lastname, username, email, bcrypt_pwd, created_at, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		user.UUID,
		user.Firstname,
		user.Lastname,
		user.Username,
		user.Email,
		user.BcryptPwd,
		user.CreatedAt,
		user.EmailVerified,
	)
	return err
}

// Validation functions
func validateUsername(username string) srvcerror.E {
	const minUsernameLength = 2
	const maxUsernameLength = 32
	reservedUsernames := map[string]struct{}{
		"admin":  {},
		"system": {},
		"test":   {},
	}
	if len(username) < minUsernameLength {
		return newErrUsernameTooShort(minUsernameLength)
	}
	if len(username) > maxUsernameLength {
		return newErrUsernameTooLong()
	}
	if _, reserved := reservedUsernames[strings.ToLower(username)]; reserved {
		return newErrUsernameReserved()
	}
	return nil
}

func validateEmail(email string) srvcerror.E {
	const maxEmailLength = 320
	if len(email) > maxEmailLength {
		return newErrEmailTooLong()
	}

	if len(email) == 0 {
		return newErrEmailEmpty()
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return newErrEmailInvalid()
	}

	return nil
}

func validatePassword(password string) srvcerror.E {
	const minPasswordLength = 8
	if len(password) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(password) > 1024 {
		return newErrPasswordTooLong()
	}
	return nil
}

func validateFirstname(firstname string) srvcerror.E {
	const maxFirstnameLength = 35
	if len(firstname) > maxFirstnameLength {
		return newErrFirstnameTooLong(maxFirstnameLength)
	}
	return nil
}

func validateLastname(lastname string) srvcerror.E {
	const maxLastnameLength = 35
	if len(lastname) > maxLastnameLength {
		return newErrLastnameTooLong(maxLastnameLength)
	}
	return nil
}
