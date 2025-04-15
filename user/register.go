package user

import (
	"context"
	"net/mail"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserParams struct {
	Username  string
	Email     string
	Firstname *string
	Lastname  *string
	Password  string
}

func (s *UserSrvc) CreateUser(ctx context.Context,
	p CreateUserParams) (res *User, err error) {

	var (
		username  Username = Username(p.Username)
		email     Email    = Email(p.Email)
		password  Password = Password(p.Password)
		firstname Firstname
		lastname  Lastname
	)

	if p.Firstname != nil {
		firstname = Firstname(*p.Firstname)
	}

	if p.Lastname != nil {
		lastname = Lastname(*p.Lastname)
	}

	for _, v := range []interface{ IsValid() error }{
		username, email, password, firstname, lastname,
	} {
		if v == nil {
			continue
		}
		if err := v.IsValid(); err != nil {
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

	row := &model.Users{
		UUID:      uuid.New(),
		Firstname: string(firstname),
		Lastname:  string(lastname),
		Username:  string(username),
		Email:     string(email),
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

func selectAllUsers(pg *sqlx.DB) ([]model.Users, error) {
	selectStmt := postgres.
		SELECT(table.Users.AllColumns).
		FROM(table.Users)

	var users []model.Users
	err := selectStmt.Query(pg, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func insertUser(pg *sqlx.DB, row *model.Users) error {
	insertStmt := table.Users.INSERT(table.Users.AllColumns).
		MODEL(row)

	_, err := insertStmt.Exec(pg)
	return err
}

type Username string

func (u Username) IsValid() error {
	const minUsernameLength = 2
	const maxUsernameLength = 32
	if len(string(u)) < minUsernameLength {
		return newErrUsernameTooShort(minUsernameLength)
	}
	if len(string(u)) > maxUsernameLength {
		return newErrUsernameTooLong()
	}
	return nil
}

func (u Username) String() string {
	return string(u)
}

type Email string

func (e Email) IsValid() error {
	const maxEmailLength = 320
	if len(string(e)) > maxEmailLength {
		return newErrEmailTooLong()
	}

	if len(string(e)) == 0 {
		return newErrEmailEmpty()
	}

	_, err := mail.ParseAddress(string(e))
	if err != nil {
		return newErrEmailInvalid()
	}

	return nil
}

func (e Email) String() string {
	return string(e)
}

type Password string

func (p Password) IsValid() error {
	const minPasswordLength = 8
	if len(string(p)) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(string(p)) > 1024 {
		return newErrPasswordTooLong()
	}
	return nil
}

type Firstname string

func (f Firstname) IsValid() error {
	const maxFirstnameLength = 35
	if len(f) > maxFirstnameLength {
		return newErrFirstnameTooLong(maxFirstnameLength)
	}
	return nil
}

type Lastname string

func (l Lastname) IsValid() error {
	const maxLastnameLength = 35
	if len(l) > maxLastnameLength {
		return newErrLastnameTooLong(maxLastnameLength)
	}
	return nil
}
