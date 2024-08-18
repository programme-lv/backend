package task

type Error struct {
	errorCode string
	msgToUser string // public
}

func (e *Error) Error() string {
	return e.msgToUser
}

func (e *Error) ErrorCode() string {
	return e.errorCode
}

const ErrTaskNotFoundCode = "task_not_found"

func newErrTaskNotFound() *Error {
	return &Error{
		errorCode: ErrTaskNotFoundCode,
		msgToUser: "uzdevums netika atrasts",
	}
}
