package subm

import "fmt"

type ErrInvalidSubmissionDetails struct {
	errorCode string
	msgToUser string
}

func (e *ErrInvalidSubmissionDetails) Error() string {
	return fmt.Sprintf("[%s]: %s", e.errorCode, e.msgToUser)
}

func (e *ErrInvalidSubmissionDetails) ErrorCode() string {
	return e.errorCode
}

func newErrInvalidSubmissionDetailsContentTooLong(maxSubmLengthKB int) error {
	return &ErrInvalidSubmissionDetails{
		errorCode: "invalid_submission_details",
		msgToUser: fmt.Sprintf("iesūtījuma kods ir pārāk garš, maksimālais garums ir %d KB", maxSubmLengthKB),
	}
}

func newErrInvalidSubmissionDetailsTaskNotFound() error {
	return &ErrInvalidSubmissionDetails{
		errorCode: "invalid_submission_details",
		msgToUser: "atbilstošais uzdevums netika atrasts",
	}
}

func newErrInvalidSubmissionDetailsUserNotFound() error {
	return &ErrInvalidSubmissionDetails{
		errorCode: "invalid_submission_details",
		msgToUser: "norādītais lietotājs netika atrasts",
	}
}

func newErrInvalidSubmissionDetailsInvalidProgrammingLanguage() error {
	return &ErrInvalidSubmissionDetails{
		errorCode: "invalid_submission_details",
		msgToUser: "nederīga programmēšanas valoda",
	}
}

type ErrInternalServerError struct {
	errorCode string
	msgToUser string
}

func (e *ErrInternalServerError) Error() string {
	return fmt.Sprintf("[%s]: %s", e.errorCode, e.msgToUser)
}

func (e *ErrInternalServerError) ErrorCode() string {
	return e.errorCode
}

func newErrInternalServerErrorGettingTask() error {
	return &ErrInternalServerError{
		errorCode: "internal_server_error",
		msgToUser: "neizdevās iegūt uzdevumu",
	}
}

type ErrUnauthorized struct {
	errorCode string
	msgToUser string
}

func (e *ErrUnauthorized) Error() string {
	return fmt.Sprintf("[%s]: %s", e.errorCode, e.msgToUser)
}

func (e *ErrUnauthorized) ErrorCode() string {
	return e.errorCode
}

func newErrUnauthorizedJwtClaimsDidNotMatchUsername() error {
	return &ErrUnauthorized{
		errorCode: "authentication_authorization_failed",
		msgToUser: "JWT norādītais lietotājvārds nesakrīt ar pieprasīto lietotājvārdu",
	}
}

func newErrInternalServerErrorGettingUser() error {
	return &ErrInternalServerError{
		errorCode: "internal_server_error",
		msgToUser: "neizdevās iegūt lietotāju",
	}
}

func newErrInternal(msg string) error {
	return &ErrInternalServerError{
		errorCode: "internal_server_error",
		msgToUser: msg,
	}
}
