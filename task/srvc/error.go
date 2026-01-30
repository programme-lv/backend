package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeTaskNotFound = "task_not_found"

func NewErrorTaskNotFound(taskId string) srvcerror.E {
	return srvcerror.New(
		ErrCodeTaskNotFound,
		fmt.Sprintf("uzdevums '%s' netika atrasts", taskId),
	).SetHttpStatusCode(http.StatusNotFound)
}

var ErrSomeTaskNotFound = srvcerror.New(
	"some_task_not_found",
	"kāds no uzdevumiem netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

const ErrCodeImageAlreadyExists = "image_already_exists"

func NewErrorImageAlreadyExists(filename string) srvcerror.E {
	return srvcerror.New(
		ErrCodeImageAlreadyExists,
		fmt.Sprintf("attēls ar nosaukumu '%s' jau eksistē", filename),
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeImageFileExtFromMimeType = "image_file_ext_from_mime_type"

func NewErrorImageFileExtFromMimeType(mime string) srvcerror.E {
	return srvcerror.New(
		ErrCodeImageFileExtFromMimeType,
		fmt.Sprintf("neizdevās iegūt attēla faila paplašinājumu [.png, .jpg, .jpeg, ...] no MIME tipa '%s'", mime),
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeGetImageWidthAndHeight = "image_width_and_height"

func NewErrorGetImageWidthAndHeight() srvcerror.E {
	return srvcerror.New(
		ErrCodeGetImageWidthAndHeight,
		"neizdevās iegūt attēla platumu un augstumu [px]",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeImageInadequateDimensions = "image_inadequate_dimensions"

func NewErrorImageInadequateDimensions() srvcerror.E {
	return srvcerror.New(
		ErrCodeImageInadequateDimensions,
		"attēls ir pārāk mazs vai pārāk liels",
	).SetHttpStatusCode(http.StatusBadRequest)
}

const ErrCodeFailedToGetTaskFromDb = "failed_to_get_task_from_db"

func NewErrorFailedToGetTaskFromDb(taskId string) srvcerror.E {
	return srvcerror.New(
		ErrCodeFailedToGetTaskFromDb,
		fmt.Sprintf("neizdevās iegūt uzdevumu '%s' no datubāzes", taskId),
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeInternalServerError = "internal_server_error"

func NewErrorInternalServerError() srvcerror.E {
	return srvcerror.New(
		ErrCodeInternalServerError,
		"iekšēja servera kļūda",
	).SetHttpStatusCode(http.StatusInternalServerError)
}

const ErrCodeTaskAlreadyExists = "task_already_exists"

func NewErrorTaskAlreadyExists(taskId string) srvcerror.E {
	return srvcerror.New(
		ErrCodeTaskAlreadyExists,
		fmt.Sprintf("uzdevums ar ID '%s' jau eksistē", taskId),
	).SetHttpStatusCode(http.StatusConflict)
}

const ErrCodeImageNotFound = "image_not_found"

func NewErrorImageNotFound(filename string) srvcerror.E {
	return srvcerror.New(
		ErrCodeImageNotFound,
		fmt.Sprintf("attēls ar nosaukumu '%s' netika atrasts", filename),
	).SetHttpStatusCode(http.StatusNotFound)
}
