package subm

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

func newErrInvalidSubmissionDetailsContentTooLong(maxSubmLengthKB int) *Error {
	return &Error{
		errorCode: "invalid_submission_details",
		msgToUser: fmt.Sprintf("iesūtījuma kods ir pārāk garš, maksimālais garums ir %d KB", maxSubmLengthKB),
	}
}

func newErrInvalidSubmissionDetailsTaskNotFound() *Error {
	return &Error{
		errorCode: "invalid_submission_details",
		msgToUser: "atbilstošais uzdevums netika atrasts",
	}
}

func newErrInvalidSubmissionDetailsUserNotFound() *Error {
	return &Error{
		errorCode: "invalid_submission_details",
		msgToUser: "norādītais lietotājs netika atrasts",
	}
}

func newErrInvalidSubmissionDetailsInvalidProgrammingLanguage() *Error {
	return &Error{
		errorCode: "invalid_submission_details",
		msgToUser: "nederīga programmēšanas valoda",
	}
}

func newErrInternalServerErrorGettingTask() *Error {
	return &Error{
		errorCode: "internal_server_error",
		msgToUser: "neizdevās iegūt uzdevumu",
	}
}

func newErrUnauthorizedJwtClaimsDidNotMatchUsername() *Error {
	return &Error{
		errorCode: "authentication_authorization_failed",
		msgToUser: "JWT norādītais lietotājvārds nesakrīt ar pieprasīto lietotājvārdu",
	}
}
