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

func newErrUsernameTooShort(minLength int) error {
	return &Error{
		errorCode: "username_too_short",
		msgToUser: "lietotājvārdam jābūt vismaz %d simbolus garam",
	}
}

func newErrUsernameTooLong() error {
	return &Error{
		errorCode: "username_too_long",
		msgToUser: "lietotājvārds ir pārāk garš",
	}
}

func newErrUsernameExists() error {
	return &Error{
		errorCode: "username_exists",
		msgToUser: "lietotājvārds jau eksistē",
	}
}

func newErrEmailExists() error {
	return &Error{
		errorCode: "email_exists",
		msgToUser: "epasts jau eksistē",
	}
}

func newErrInternalServerError() error {
	return &Error{
		errorCode: "internal_server_error",
		msgToUser: "iekšēja servera kļūda",
	}
}

func newErrEmailTooLong() error {
	return &Error{
		errorCode: "email_too_long",
		msgToUser: "epasts ir pārāk garš",
	}
}

func newErrEmailEmpty() error {
	return &Error{
		errorCode: "email_empty",
		msgToUser: "epasts nedrīkst būt tukšs",
	}
}

func newErrEmailInvalid() error {
	return &Error{
		errorCode: "email_invalid",
		msgToUser: "epasts ir nederīgs",
	}
}

func newErrPasswordTooShort(minLength int) error {
	return &Error{
		errorCode: "password_too_short",
		msgToUser: "parolei jābūt vismaz %d simbolus garai",
	}
}

func newErrPasswordTooLong() error {
	return &Error{
		errorCode: "password_too_long",
		msgToUser: "parole ir pārāk gara",
	}
}

func newErrFirstnameTooLong(maxLength int) error {
	return &Error{
		errorCode: "firstname_too_long",
		msgToUser: fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength),
	}
}

func newErrLastnameTooLong(maxLength int) error {
	return &Error{
		errorCode: "lastname_too_long",
		msgToUser: fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength),
	}
}
