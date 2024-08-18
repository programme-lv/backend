package user

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerr"
)

const ErrCodeUsernameTooShort = "username_too_short"

func newErrUsernameTooShort(minLength int) *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUsernameTooShort,
		fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameTooLong = "username_too_long"

func newErrUsernameTooLong() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUsernameTooLong,
		"lietotājvārds ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameAlreadyExists = "username_exists"

func newErrUsernameExists() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUsernameAlreadyExists,
		"lietotājvārds jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeEmailAlreadyExists = "email_exists"

func newErrEmailExists() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeEmailAlreadyExists,
		"epasts jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeInternalServerError = "internal_server_error"

func newErrInternalServerError() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeEmailTooLong = "email_too_long"

func newErrEmailTooLong() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeEmailTooLong,
		"epasts ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeEmailEmpty = "email_empty"

func newErrEmailEmpty() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeEmailEmpty,
		"epasts nedrīkst būt tukšs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordEmpty = "password_empty"

func newErrEmailInvalid() *srvcerr.Error {
	return srvcerr.New(
		ErrCodePasswordEmpty,
		"epasts ir nederīgs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooShort = "password_too_short"

func newErrPasswordTooShort(minLength int) *srvcerr.Error {
	return srvcerr.New(
		ErrCodePasswordTooShort,
		fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooLong = "password_too_long"

func newErrPasswordTooLong() *srvcerr.Error {
	return srvcerr.New(
		ErrCodePasswordTooLong,
		"parole ir pārāk gara",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeFirstnameTooLong = "firstname_too_long"

func newErrFirstnameTooLong(maxLength int) *srvcerr.Error {
	return srvcerr.New(
		ErrCodeFirstnameTooLong,
		fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeLastnameTooLong = "lastname_too_long"

func newErrLastnameTooLong(maxLength int) *srvcerr.Error {
	return srvcerr.New(
		ErrCodeLastnameTooLong,
		fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUserNotFound = "user_not_found"

func newErrUserNotFound() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUserNotFound,
		"lietotājs netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeUsernameOrPasswordIncorrect = "username_or_password_incorrect"

func newErrUsernameOrPasswordIncorrect() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUsernameOrPasswordIncorrect,
		"lietotājvārds vai parole nav pareiza",
	).SetHttpStatusCode(http.StatusUnauthorized)
}
