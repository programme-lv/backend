package task

import "fmt"

type ErrTaskNotFound struct {
	errorCode string
	msgToUser string
}

func (e *ErrTaskNotFound) Error() string {
	return fmt.Sprintf("[%s]: %s", e.errorCode, e.msgToUser)
}

func (e *ErrTaskNotFound) ErrorCode() string {
	return e.errorCode
}

func newErrTaskNotFound() error {
	return &ErrTaskNotFound{
		errorCode: "task_not_found",
		msgToUser: "uzdevums netika atrasts",
	}
}
