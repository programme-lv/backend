package user

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeUsernameTooShort = "username_too_short"

func newErrUsernameTooShort(minLength int) srvcerror.E {
	return srvcerror.New(
		ErrCodeUsernameTooShort,
		fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameTooLong = "username_too_long"

func newErrUsernameTooLong() srvcerror.E {
	return srvcerror.New(
		ErrCodeUsernameTooLong,
		"lietotājvārds ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUsernameAlreadyExists = "username_exists"

func newErrUsernameExists() srvcerror.E {
	return srvcerror.New(
		ErrCodeUsernameAlreadyExists,
		"lietotājvārds jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeEmailAlreadyExists = "email_exists"

var ErrEmailAlreadyExists = srvcerror.New(
	ErrCodeEmailAlreadyExists,
	"epasts jau eksistē",
).SetHttpStatusCode(http.StatusConflict)

func newErrEmailExists() srvcerror.E {
	return srvcerror.New(
		ErrCodeEmailAlreadyExists,
		"epasts jau eksistē",
	).SetHttpStatusCode(http.StatusConflict)
}

func newErrInternalSE() srvcerror.E {
	return srvcerror.InternalServerError()
}

const ErrCodeEmailTooLong = "email_too_long"

func newErrEmailTooLong() srvcerror.E {
	return srvcerror.New(
		ErrCodeEmailTooLong,
		"epasts ir pārāk garš",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeEmailEmpty = "email_empty"

func newErrEmailEmpty() srvcerror.E {
	return srvcerror.New(
		ErrCodeEmailEmpty,
		"epasts nedrīkst būt tukšs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordEmpty = "password_empty"

func newErrEmailInvalid() srvcerror.E {
	return srvcerror.New(
		ErrCodePasswordEmpty,
		"epasts ir nederīgs",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooShort = "password_too_short"

func newErrPasswordTooShort(minLength int) srvcerror.E {
	return srvcerror.New(
		ErrCodePasswordTooShort,
		fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodePasswordTooLong = "password_too_long"

func newErrPasswordTooLong() srvcerror.E {
	return srvcerror.New(
		ErrCodePasswordTooLong,
		"parole ir pārāk gara",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeFirstnameTooLong = "firstname_too_long"

func newErrFirstnameTooLong(maxLength int) srvcerror.E {
	return srvcerror.New(
		ErrCodeFirstnameTooLong,
		fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeLastnameTooLong = "lastname_too_long"

func newErrLastnameTooLong(maxLength int) srvcerror.E {
	return srvcerror.New(
		ErrCodeLastnameTooLong,
		fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUserNotFound = "user_not_found"

var ErrUserNotFound = srvcerror.New(
	ErrCodeUserNotFound,
	"lietotājs netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

const ErrCodeUsernameOrPasswordIncorrect = "username_or_password_incorrect"

func newErrUsernameOrPasswordIncorrect() srvcerror.E {
	return srvcerror.New(
		ErrCodeUsernameOrPasswordIncorrect,
		"lietotājvārds vai parole nav pareiza",
	).SetHttpStatusCode(http.StatusUnauthorized)
}
