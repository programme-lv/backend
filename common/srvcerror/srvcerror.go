package srvcerror

import (
	"errors"
	"net/http"
)

type E interface {
	error
	Is(err error) bool

	ErrorCode() string // unique identifier for the type of underlying cause

	HttpStatusCode() int
	SetHttpStatusCode(code int) E
	WithMsg(msg string) E
}

// Error implements the E interface
type Error struct {
	errorCode string

	msgToUser string // public

	httpStatus int // optional, for HTTP responses
}

func (e Error) Is(err error) bool {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		if svcErr.errorCode == e.errorCode {
			return true
		}
	}
	var svcErr2 Error
	if errors.As(err, &svcErr2) {
		if svcErr2.errorCode == e.errorCode {
			return true
		}
	}
	return false
}

func Is(err error, code string) bool {
	if err == nil {
		return false
	}
	var svcErr E
	if errors.As(err, &svcErr) {
		return svcErr.ErrorCode() == code
	}
	return false
}

func (e Error) Error() string {
	return e.msgToUser
}

func (e Error) ErrorCode() string {
	return e.errorCode
}

func (e Error) HttpStatusCode() int {
	if e.httpStatus == 0 {
		return http.StatusInternalServerError
	}
	return e.httpStatus
}

func (e Error) SetHttpStatusCode(code int) E {
	e.httpStatus = code
	return e
}

func (e Error) WithMsg(msg string) E {
	e.msgToUser = msg
	return e
}

func New(errorCode string, msgToUser string) E {
	return Error{
		errorCode: errorCode,
		msgToUser: msgToUser,
	}
}

const ErrCodeInternalServerError = "internal_server_error"

var ErrInternalServerError = New(
	ErrCodeInternalServerError,
	"iekšēja servera kļūda",
).SetHttpStatusCode(http.StatusInternalServerError)

var ErrInternal = ErrInternalServerError

func InternalServerError() E {
	return ErrInternal
}
