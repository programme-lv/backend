package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	"goa.design/clue/log"
	"golang.org/x/crypto/bcrypt"
)

type Validatable interface {
	IsValid() error
}

type Username struct {
	value string
}

func (u *Username) IsValid() error {
	const minUsernameLength = 2
	const maxUsernameLength = 32
	if len(u.value) < minUsernameLength {
		return newErrUsernameTooShort(minUsernameLength)
	}
	if len(u.value) > maxUsernameLength {
		return newErrUsernameTooLong()
	}
	return nil
}

func (u *Username) String() string {
	return u.value
}

type Email struct {
	Value string
}

func (e *Email) IsValid() error {
	const maxEmailLength = 320
	if len(e.Value) > maxEmailLength {
		return newErrEmailTooLong()
	}

	if len(e.Value) == 0 {
		return newErrEmailEmpty()
	}

	_, err := mail.ParseAddress(e.Value)
	if err != nil {
		return newErrEmailInvalid()
	}

	return nil
}

func (e *Email) String() string {
	return e.Value
}

type Password struct {
	Value string
}

func (p *Password) IsValid() error {
	const minPasswordLength = 8
	if len(p.Value) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(p.Value) > 1024 {
		return newErrPasswordTooLong()
	}
	return nil
}

type Firstname struct {
	Value *string
}

func (f *Firstname) IsValid() error {
	if f.Value == nil {
		return nil
	}

	const maxFirstnameLength = 35
	if len(*f.Value) > maxFirstnameLength {
		return newErrFirstnameTooLong(maxFirstnameLength)
	}
	return nil
}

type Lastname struct {
	Value *string
}

func (l *Lastname) IsValid() error {
	if l.Value == nil {
		return nil
	}
	const maxLastnameLength = 35
	if len(*l.Value) > maxLastnameLength {
		return newErrLastnameTooLong(maxLastnameLength)
	}
	return nil
}

func (s *UserService) CreateUser(ctx context.Context, p *UserPayload) (res *User, err error) {
	username := Username{p.Username}
	email := Email{p.Email}
	password := Password{p.Password}
	firstname := Firstname{p.Firstname}
	lastname := Lastname{p.Lastname}

	for _, v := range []Validatable{&username, &email, &password, &firstname, &lastname} {
		err := v.IsValid()
		if err != nil {
			return nil, err
		}
	}

	allUsers, err := s.ddbUserTable.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range allUsers {
		if user.Username == p.Username {
			errRes := newErrUsernameExists()
			errRes.SetDebugInfo(fmt.Errorf("username %s already exists", p.Username))
			return nil, errRes
		}
		if user.Email == p.Email {
			errRes := newErrEmailExists()
			errRes.SetDebugInfo(fmt.Errorf("email %s already exists", p.Email))
			return nil, errRes
		}
	}

	bcryptPwd, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("error hashing password")
	}

	uuid := uuid.New()

	row := &UserRow{
		Uuid:      uuid.String(),
		Username:  username.String(),
		Email:     email.String(),
		BcryptPwd: string(bcryptPwd),
		Firstname: firstname.Value,
		Lastname:  lastname.Value,
		Version:   0,
		CreatedAt: time.Now(),
	}

	err = s.ddbUserTable.Save(ctx, row)
	if err != nil {
		// TODO: automatically retry with exponential backoff on version conflict
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, newErrInternalServerError()
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, newErrInternalServerError()
		}
	}

	res = &User{
		UUID:      uuid.String(),
		Username:  p.Username,
		Email:     p.Email,
		Firstname: p.Firstname,
		Lastname:  p.Lastname,
	}

	return res, nil
}
