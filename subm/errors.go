package subm

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerr"
)

const ErrCodeSubmissionTooLong = "submission_too_long"

func newErrSubmissionTooLong(maxSubmLengthKB int) *srvcerr.Error {
	return srvcerr.New(
		ErrCodeSubmissionTooLong,
		fmt.Sprintf("Iesūtījuma kods ir pārāk garš, maksimālais garums ir %d KB", maxSubmLengthKB),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeTaskNotFound = "task_not_found"

func newErrTaskNotFound() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeTaskNotFound,
		"Atbilstošais uzdevums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeUserNotFound = "user_not_found"

func newErrUserNotFound() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUserNotFound,
		"Norādītais lietotājs netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeInvalidProgLang = "invalid_programming_language"

func newErrInvalidProgLang() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeInvalidProgLang,
		"Nederīga programmēšanas valoda",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUnauthorized = "unauthorized_access"

func newErrUnauthorizedUsernameMismatch() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUnauthorized,
		"JWT norādītais lietotājvārds nesakrīt ar pieprasīto lietotājvārdu",
	).SetHttpStatusCode(http.StatusUnauthorized)
}

func newErrJwtTokenMissing() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeUnauthorized,
		"JWT netika atrasts",
	).SetHttpStatusCode(http.StatusUnauthorized)
}

const ErrCodeSubmissionNotFound = "submission_not_found"

func newErrSubmissionNotFound() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeSubmissionNotFound,
		"Atbilstošais iesūtījums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}