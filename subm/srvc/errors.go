package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeSubmissionTooLong = "submission_too_long"

func ErrSubmissionTooLong(maxSubmLengthKB int) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeSubmissionTooLong,
		fmt.Sprintf("Maksimālais koda garums ir %d KB.", maxSubmLengthKB),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeUserNotFound = "user_not_found"

func ErrUserNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeUserNotFound,
		"Norādītais lietotājs netika atrasts.",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeInvalidProgLang = "invalid_programming_language"

func ErrInvalidProgLang(langId string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInvalidProgLang,
		fmt.Sprintf("Programmēšanas valoda: %s", langId),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeSubmissionNotFound = "submission_not_found"

func ErrSubmissionNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeSubmissionNotFound,
		"Atbilstošais iesūtījums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}
