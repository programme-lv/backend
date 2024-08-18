package user

import "fmt"

type Error struct {
	errorCode  string
	msgToUser  string // public
	dbgInfoErr error  // private, for debugging
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

const ErrUsernameTooShortCode = "username_too_short"

func newErrUsernameTooShort(minLength int) *Error {
	return &Error{
		errorCode: ErrUsernameTooShortCode,
		msgToUser: fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength),
	}
}

func newErrUsernameTooLong() *Error {
	return &Error{
		errorCode: "username_too_long",
		msgToUser: "lietotājvārds ir pārāk garš",
	}
}

func newErrUsernameExists() *Error {
	return &Error{
		errorCode: "username_exists",
		msgToUser: "lietotājvārds jau eksistē",
	}
}

func newErrEmailExists() *Error {
	return &Error{
		errorCode: "email_exists",
		msgToUser: "epasts jau eksistē",
	}
}

func newErrInternalServerError() *Error {
	return &Error{
		errorCode: "internal_server_error",
		msgToUser: "iekšēja servera kļūda",
	}
}

func newErrEmailTooLong() *Error {
	return &Error{
		errorCode: "email_too_long",
		msgToUser: "epasts ir pārāk garš",
	}
}

func newErrEmailEmpty() *Error {
	return &Error{
		errorCode: "email_empty",
		msgToUser: "epasts nedrīkst būt tukšs",
	}
}

func newErrEmailInvalid() *Error {
	return &Error{
		errorCode: "email_invalid",
		msgToUser: "epasts ir nederīgs",
	}
}

func newErrPasswordTooShort(minLength int) *Error {
	return &Error{
		errorCode: "password_too_short",
		msgToUser: fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength),
	}
}

func newErrPasswordTooLong() *Error {
	return &Error{
		errorCode: "password_too_long",
		msgToUser: "parole ir pārāk gara",
	}
}

func newErrFirstnameTooLong(maxLength int) *Error {
	return &Error{
		errorCode: "firstname_too_long",
		msgToUser: fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
	}
}

func newErrLastnameTooLong(maxLength int) *Error {
	return &Error{
		errorCode: "lastname_too_long",
		msgToUser: fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
	}
}

const ErrUserNotFoundCode = "user_not_found"

func newErrUserNotFound() *Error {
	return &Error{
		errorCode: ErrUserNotFoundCode,
		msgToUser: "lietotājs netika atrasts",
	}
}

func newErrUsernameOrPasswordIncorrect() *Error {
	return &Error{
		errorCode: "username_or_password_incorrect",
		msgToUser: "lietotājvārds vai parole nav pareiza",
	}
}
