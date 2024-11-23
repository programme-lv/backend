package srvcerror

import "net/http"

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

func (e *Error) SetDebug(err error) *Error {
	e.dbgInfoErr = err
	return e
}

func (e *Error) HttpStatusCode() int {
	if e.httpStatus == 0 {
		return http.StatusInternalServerError
	}
	return e.httpStatus
}

func (e *Error) SetHttpStatusCode(code int) *Error {
	e.httpStatus = code
	return e
}

func New(errorCode string, msgToUser string) *Error {
	return &Error{
		errorCode: errorCode,
		msgToUser: msgToUser,
	}
}

const ErrCodeInternalServerError = "internal_server_error"

func ErrInternalSE() *Error {
	return New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}
