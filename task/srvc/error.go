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
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrSomeTaskNotFound = "some_task_not_found"

func NewErrorSomeTaskNotFound() *srvcerror.Error {
	return srvcerror.New(
		ErrSomeTaskNotFound,
		"daži uzdevumi netika atrasti",
	).SetHttpStatusCode(http.StatusNotFound)
}

const ErrCodeImageAlreadyExists = "image_already_exists"

func NewErrorImageAlreadyExists(filename string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageAlreadyExists,
		fmt.Sprintf("attēls ar nosaukumu '%s' jau eksistē", filename),
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeImageFileExtFromMimeType = "image_file_ext_from_mime_type"

func NewErrorImageFileExtFromMimeType(mime string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageFileExtFromMimeType,
		fmt.Sprintf("neizdevās iegūt attēla faila paplašinājumu [.png, .jpg, .jpeg, ...] no MIME tipa '%s'", mime),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeGetImageWidthAndHeight = "image_width_and_height"

func NewErrorGetImageWidthAndHeight() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeGetImageWidthAndHeight,
		"neizdevās iegūt attēla platumu un augstumu [px]",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeImageInadequateDimensions = "image_inadequate_dimensions"

func NewErrorImageInadequateDimensions() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageInadequateDimensions,
		"attēls ir pārāk mazs vai pārāk liels",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeFailedToGetTaskFromDb = "failed_to_get_task_from_db"

func NewErrorFailedToGetTaskFromDb(taskId string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeFailedToGetTaskFromDb,
		fmt.Sprintf("neizdevās iegūt uzdevumu '%s' no datubāzes", taskId),
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeInternalServerError = "internal_server_error"

func NewErrorInternalServerError() *srvcerror.Error {
	return srvcerror.New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeTaskAlreadyExists = "task_already_exists"

func NewErrorTaskAlreadyExists(taskId string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeTaskAlreadyExists,
		fmt.Sprintf("uzdevums ar ID '%s' jau eksistē", taskId),
	).SetHttpStatusCode(http.StatusConflict)
}
