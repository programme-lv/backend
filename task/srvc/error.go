package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeTaskNotFound = "task_not_found"

func NewErrorTaskNotFound(taskId string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeTaskNotFound,
		fmt.Sprintf("uzdevums '%s' netika atrasts", taskId),
	)
}

const ErrCodeImageAlreadyExists = "image_already_exists"

func NewErrorImageAlreadyExists(filename string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageAlreadyExists,
		fmt.Sprintf("attēls ar nosaukumu '%s' jau eksistē", filename),
	)
}

const ErrCodeImageFileExtFromMimeType = "image_file_ext_from_mime_type"

func NewErrorImageFileExtFromMimeType(mime string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageFileExtFromMimeType,
		fmt.Sprintf("neizdevās iegūt attēla faila paplašinājumu [.png, .jpg, .jpeg, ...] no MIME tipa '%s'", mime),
	)
}

const ErrCodeGetImageWidthAndHeight = "image_width_and_height"

func NewErrorGetImageWidthAndHeight() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeGetImageWidthAndHeight,
		"neizdevās iegūt attēla platumu un augstumu [px]",
	)
}

const ErrCodeImageInadequateDimensions = "image_inadequate_dimensions"

func NewErrorImageInadequateDimensions() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageInadequateDimensions,
		"attēls ir pārāk mazs vai pārāk liels",
	)
}

const ErrCodeFailedToGetTaskFromDb = "failed_to_get_task_from_db"

func NewErrorFailedToGetTaskFromDb(taskId string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeFailedToGetTaskFromDb,
		fmt.Sprintf("neizdevās iegūt uzdevumu '%s' no datubāzes", taskId),
	)
}

const ErrCodeInternalServerError = "internal_server_error"

func NewErrorInternalServerError() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}
