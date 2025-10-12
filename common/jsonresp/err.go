package jsonresp

import "net/http"

type HttpError struct {
	ErrCode    string // short well-formatted identifier
	ErrMsg     string
	HttpStatus int
}

func (e HttpError) Error() string {
	return e.ErrMsg
}

func (e HttpError) HttpStatusCode() int {
	return e.HttpStatus
}

func (e HttpError) ErrorCode() string {
	return e.ErrCode
}

func (e HttpError) Is(err error) bool {
	if other, ok := err.(HttpError); ok {
		return other.ErrCode == e.ErrCode
	}
	if other, ok := err.(*HttpError); ok {
		return other.ErrCode == e.ErrCode
	}
	return false
}

func (e HttpError) WithMsg(errMsg string) HttpError {
	return HttpError{
		ErrCode:    e.ErrCode,
		ErrMsg:     errMsg,
		HttpStatus: e.HttpStatus,
	}
}

var (
	ErrInternalServerError = HttpError{
		ErrCode:    "internal_server_error",
		ErrMsg:     "internal server error; please contact the administrator",
		HttpStatus: http.StatusInternalServerError,
	}
	ErrHttpBadRequest = HttpError{
		ErrCode:    "http_bad_request",
		ErrMsg:     "bad request",
		HttpStatus: http.StatusBadRequest,
	}
	ErrHttpForbidden = HttpError{
		ErrCode:    "http_forbidden",
		ErrMsg:     "forbidden",
		HttpStatus: http.StatusForbidden,
	}
	ErrHttpUnauthorized = HttpError{
		ErrCode:    "http_unauthorized",
		ErrMsg:     "unauthorized",
		HttpStatus: http.StatusUnauthorized,
	}
)
