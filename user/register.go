package user

import (
	"context"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserParams struct {
	Username  string
	Email     string
	Firstname *string
	Lastname  *string
	Password  string
}

func (s *UserSrvc) CreateUser(ctx context.Context, p CreateUserParams) (res *User, err error) {
	// Validate all fields
	if err := validateUsername(p.Username); err != nil {
		return nil, err
	}
	if err := validateEmail(p.Email); err != nil {
		return nil, err
	}
	if err := validatePassword(p.Password); err != nil {
		return nil, err
	}
	if p.Firstname != nil {
		if err := validateFirstname(*p.Firstname); err != nil {
			return nil, err
		}
	}
	if p.Lastname != nil {
		if err := validateLastname(*p.Lastname); err != nil {
			return nil, err
		}
	}

	all, err := selectAllUsers(s.postgres)
	if err != nil {
		return nil, err
	}

	for _, user := range all {
		// username must be unique
		if user.Username == p.Username {
			return nil, newErrUsernameExists()
		}
		// email must be unique
		if user.Email == p.Email {
			return nil, newErrEmailExists()
		}
	}

	bcryptPwd, err := bcrypt.GenerateFromPassword(
		[]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, newErrInternalSE().SetDebug(err)
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
		UUID:      uuid.New(),
		Firstname: firstname,
		Lastname:  lastname,
		Username:  p.Username,
		Email:     p.Email,
		BcryptPwd: string(bcryptPwd),
		CreatedAt: time.Now(),
	}

	err = insertUser(s.postgres, row)
	if err != nil {
		return nil, newErrInternalSE().SetDebug(err)
	}

	res = &User{
		UUID:      row.UUID,
		Username:  row.Username,
		Email:     row.Email,
		Firstname: &row.Firstname,
		Lastname:  &row.Lastname,
	}

	return res, nil
}

type dbUser struct {
	UUID      uuid.UUID
	Firstname string
	Lastname  string
	Username  string
	Email     string
	BcryptPwd string
	CreatedAt time.Time
}

func selectAllUsers(pg *pgxpool.Pool) ([]dbUser, error) {
	rows, err := pg.Query(context.Background(), `
		SELECT uuid, firstname, lastname, username, email, bcrypt_pwd, created_at
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
		INSERT INTO users (uuid, firstname, lastname, username, email, bcrypt_pwd, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		user.UUID,
		user.Firstname,
		user.Lastname,
		user.Username,
		user.Email,
		user.BcryptPwd,
		user.CreatedAt,
	)
	return err
}

// Validation functions
func validateUsername(username string) error {
	const minUsernameLength = 2
	const maxUsernameLength = 32
	if len(username) < minUsernameLength {
		return newErrUsernameTooShort(minUsernameLength)
	}
	if len(username) > maxUsernameLength {
		return newErrUsernameTooLong()
	}
	return nil
}

func validateEmail(email string) error {
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

func validatePassword(password string) error {
	const minPasswordLength = 8
	if len(password) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(password) > 1024 {
		return newErrPasswordTooLong()
	}
	return nil
}

func validateFirstname(firstname string) error {
	const maxFirstnameLength = 35
	if len(firstname) > maxFirstnameLength {
		return newErrFirstnameTooLong(maxFirstnameLength)
	}
	return nil
}

func validateLastname(lastname string) error {
	const maxLastnameLength = 35
	if len(lastname) > maxLastnameLength {
		return newErrLastnameTooLong(maxLastnameLength)
	}
	return nil
}
