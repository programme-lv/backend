package submsrvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeSubmissionTooLong = "submission_too_long"

func ErrSubmissionTooLong(maxSubmLengthKB int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeSubmissionTooLong,
		fmt.Sprintf("Iesūtījuma kods ir pārāk garš, maksimālais garums ir %d KB", maxSubmLengthKB),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeTaskNotFound = "task_not_found"

func ErrTaskNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeTaskNotFound,
		"Atbilstošais uzdevums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeUserNotFound = "user_not_found"

func ErrUserNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUserNotFound,
		"Norādītais lietotājs netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeInvalidProgLang = "invalid_programming_language"

func ErrInvalidProgLang() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidProgLang,
		"Nederīga programmēšanas valoda",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUnauthorized = "unauthorized_access"

func ErrUnauthorizedUsernameMismatch() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUnauthorized,
		"JWT norādītais lietotājvārds nesakrīt ar pieprasīto lietotājvārdu",
	).SetHttpStatusCode(http.StatusUnauthorized)
}

func ErrJwtTokenMissing() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUnauthorized,
		"JWT netika atrasts",
	).SetHttpStatusCode(http.StatusUnauthorized)
}

const ErrCodeSubmissionNotFound = "submission_not_found"

func ErrSubmissionNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeSubmissionNotFound,
		"Atbilstošais iesūtījums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}

func ErrInternalSE() *srvcerror.Error {
	return srvcerror.ErrInternalSE()
}
