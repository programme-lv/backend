package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	usergen "github.com/programme-lv/backend/gen/users"
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
		return usergen.InvalidUserDetails(fmt.Sprintf(
			"lietotājvārdam jābūt vismaz %d simbolus garam",
			minUsernameLength,
		))
	}
	if len(u.value) > maxUsernameLength {
		return usergen.InvalidUserDetails(
			"lietotājvārds ir pārāk garš",
		)
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
		return usergen.InvalidUserDetails(
			"epasts ir pārāk garš",
		)
	}

	if len(e.Value) == 0 {
		return usergen.InvalidUserDetails("epasts ir obligāts")
	}

	_, err := mail.ParseAddress(e.Value)
	if err != nil {
		return usergen.InvalidUserDetails("epasts nav derīgs")
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
		return usergen.InvalidUserDetails(fmt.Sprintf(
			"parolei jābūt vismaz %d simbolus garai",
			minPasswordLength,
		))
	}
	if len(p.Value) > 1024 {
		return usergen.InvalidUserDetails("parole ir neadekvāti gara")
	}
	return nil
}

type Firstname struct {
	Value string
}

func (f *Firstname) IsValid() error {
	const maxFirstnameLength = 35
	if len(f.Value) > maxFirstnameLength {
		return usergen.InvalidUserDetails(fmt.Sprintf(
			"vārds nedrīkst būt garāks par %d simboliem",
			maxFirstnameLength,
		))
	}
	return nil
}

type Lastname struct {
	Value string
}

func (l *Lastname) IsValid() error {
	const maxLastnameLength = 35
	if len(l.Value) > maxLastnameLength {
		return usergen.InvalidUserDetails(fmt.Sprintf(
			"uzvārds nedrīkst būt garāks par %d simboliem",
			maxLastnameLength,
		))
	}
	return nil
}

func (s *userssrvc) CreateUser(ctx context.Context, p *usergen.UserPayload) (res *usergen.User, err error) {
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
			return nil, usergen.UsernameExistsConflict(
				fmt.Sprintf("lietotājvārds %s jau eksistē", p.Username),
			)
		}
		if user.Email == p.Email {
			return nil, usergen.EmailExistsConflict(
				fmt.Sprintf("epasts %s jau eksistē", p.Email),
			)
		}
	}

	bcryptPwd, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("error hashing password")
	}

	uuid := uuid.New()

	var firstnamePtr *string = nil
	if p.Firstname != "" {
		firstnamePtr = &firstname.Value
	}

	var lastnamePtr *string = nil
	if p.Lastname != "" {
		lastnamePtr = &lastname.Value
	}

	row := &UserRow{
		Uuid:      uuid.String(),
		Username:  username.String(),
		Email:     email.String(),
		BcryptPwd: bcryptPwd,
		Firstname: firstnamePtr,
		Lastname:  lastnamePtr,
		Version:   0,
		CreatedAt: time.Now(),
	}

	err = s.ddbUserTable.Save(ctx, row)
	if err != nil {
		// TODO: automatically retry with exponential backoff on version conflict
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, usergen.InternalError("version conflict saving user")
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, usergen.InternalError("error saving user")
		}
	}

	res = &usergen.User{
		UUID:      uuid.String(),
		Username:  p.Username,
		Email:     p.Email,
		Firstname: p.Firstname,
		Lastname:  p.Lastname,
	}

	return res, nil
}
