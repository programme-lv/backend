package usersrvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeUsernameTooShort = "username_too_short"

func newErrUsernameTooShort(minLength int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUsernameTooShort,
		fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameTooLong = "username_too_long"

func newErrUsernameTooLong() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUsernameTooLong,
		"lietotājvārds ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameAlreadyExists = "username_exists"

func newErrUsernameExists() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUsernameAlreadyExists,
		"lietotājvārds jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeEmailAlreadyExists = "email_exists"

func newErrEmailExists() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeEmailAlreadyExists,
		"epasts jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeInternalServerError = "internal_server_error"

func newErrInternalSE() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeEmailTooLong = "email_too_long"

func newErrEmailTooLong() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeEmailTooLong,
		"epasts ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeEmailEmpty = "email_empty"

func newErrEmailEmpty() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeEmailEmpty,
		"epasts nedrīkst būt tukšs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordEmpty = "password_empty"

func newErrEmailInvalid() *srvcerror.Error {
	return srvcerror.New(
		ErrCodePasswordEmpty,
		"epasts ir nederīgs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooShort = "password_too_short"

func newErrPasswordTooShort(minLength int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodePasswordTooShort,
		fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooLong = "password_too_long"

func newErrPasswordTooLong() *srvcerror.Error {
	return srvcerror.New(
		ErrCodePasswordTooLong,
		"parole ir pārāk gara",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeFirstnameTooLong = "firstname_too_long"

func newErrFirstnameTooLong(maxLength int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeFirstnameTooLong,
		fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeLastnameTooLong = "lastname_too_long"

func newErrLastnameTooLong(maxLength int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeLastnameTooLong,
		fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUserNotFound = "user_not_found"

func newErrUserNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUserNotFound,
		"lietotājs netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeUsernameOrPasswordIncorrect = "username_or_password_incorrect"

func newErrUsernameOrPasswordIncorrect() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUsernameOrPasswordIncorrect,
		"lietotājvārds vai parole nav pareiza",
	).SetHttpStatusCode(http.StatusUnauthorized)
}
