package user

import (
	"fmt"
	"net/http"
)

type Error struct {
	errorCode  string
	msgToUser  string // public
	dbgInfoErr error  // private, for debugging

	httpStatus int // optional, for HTTP responses
}

func (e *Error) Error() string {
	return e.msgToUser
}

func (e *Error) ErrorCode() string {
	return e.errorCode
}

func (e *Error) DebugInfo() error {
	return e.dbgInfoErr
}

func (e *Error) SetDebugInfo(err error) {
	e.dbgInfoErr = err
}

func (e *Error) HttpStatusCode() int {
	if e.httpStatus == 0 {
		return http.StatusInternalServerError
	}
	return e.httpStatus
}

func (e *Error) SetHttpStatusCode(code int) {
	e.httpStatus = code
}

const ErrCodeUsernameTooShort = "username_too_short"

func newErrUsernameTooShort(minLength int) *Error {
	return &Error{
		errorCode:  ErrCodeUsernameTooShort,
		msgToUser:  fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength),
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeUsernameTooLong = "username_too_long"

func newErrUsernameTooLong() *Error {
	return &Error{
		errorCode:  "username_too_long",
		msgToUser:  "lietotājvārds ir pārāk garš",
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeUsernameAlreadyExists = "username_exists"

func newErrUsernameExists() *Error {
	return &Error{
		errorCode:  ErrCodeUsernameAlreadyExists,
		msgToUser:  "lietotājvārds jau eksistē",
		httpStatus: http.StatusConflict,
	}
}

const ErrCodeEmailAlreadyExists = "email_exists"

func newErrEmailExists() *Error {
	return &Error{
		errorCode:  ErrCodeEmailAlreadyExists,
		msgToUser:  "epasts jau eksistē",
		httpStatus: http.StatusConflict,
	}
}

const ErrCodeInternalServerError = "internal_server_error"

func newErrInternalServerError() *Error {
	return &Error{
		errorCode:  ErrCodeInternalServerError,
		msgToUser:  "iekšēja servera kļūda",
		httpStatus: http.StatusInternalServerError,
	}
}

const ErrCodeEmailTooLong = "email_too_long"

func newErrEmailTooLong() *Error {
	return &Error{
		errorCode:  ErrCodeEmailTooLong,
		msgToUser:  "epasts ir pārāk garš",
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeEmailEmpty = "email_empty"

func newErrEmailEmpty() *Error {
	return &Error{
		errorCode:  ErrCodeEmailEmpty,
		msgToUser:  "epasts nedrīkst būt tukšs",
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodePasswordEmpty = "password_empty"

func newErrEmailInvalid() *Error {
	return &Error{
		errorCode:  ErrCodePasswordEmpty,
		msgToUser:  "epasts ir nederīgs",
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodePasswordTooShort = "password_too_short"

func newErrPasswordTooShort(minLength int) *Error {
	return &Error{
		errorCode:  ErrCodePasswordTooShort,
		msgToUser:  fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength),
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodePasswordTooLong = "password_too_long"

func newErrPasswordTooLong() *Error {
	return &Error{
		errorCode:  ErrCodePasswordTooLong,
		msgToUser:  "parole ir pārāk gara",
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeFirstnameTooLong = "firstname_too_long"

func newErrFirstnameTooLong(maxLength int) *Error {
	return &Error{
		errorCode:  ErrCodeFirstnameTooLong,
		msgToUser:  fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeLastnameTooLong = "lastname_too_long"

func newErrLastnameTooLong(maxLength int) *Error {
	return &Error{
		errorCode:  ErrCodeLastnameTooLong,
		msgToUser:  fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
		httpStatus: http.StatusBadRequest,
	}
}

const ErrCodeUserNotFound = "user_not_found"

func newErrUserNotFound() *Error {
	return &Error{
		errorCode:  ErrCodeUserNotFound,
		msgToUser:  "lietotājs netika atrasts",
		httpStatus: http.StatusNotFound,
	}
}

const ErrCodeUsernameOrPasswordIncorrect = "username_or_password_incorrect"

func newErrUsernameOrPasswordIncorrect() *Error {
	return &Error{
		errorCode:  ErrCodeUsernameOrPasswordIncorrect,
		msgToUser:  "lietotājvārds vai parole nav pareiza",
		httpStatus: http.StatusUnauthorized,
	}
}
