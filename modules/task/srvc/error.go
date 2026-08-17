package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

var ErrTaskNotFound = srvcerror.New(
	"task_not_found",
	"uzdevums netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

func errTaskNotFound(taskId string) srvcerror.E {
	return ErrTaskNotFound.WithMsg(fmt.Sprintf("uzdevums '%s' netika atrasts", taskId))
}

// ErrSomeTaskNotFound means a batch name lookup missed at least one short ID.
var ErrSomeTaskNotFound = srvcerror.New(
	"some_task_not_found",
	"kāds no uzdevumiem netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

var ErrImageAlreadyExists = srvcerror.New(
	"image_already_exists",
	"attēls ar šādu nosaukumu jau eksistē",
).SetHttpStatusCode(http.StatusConflict)

func errImageAlreadyExists(filename string) srvcerror.E {
	return ErrImageAlreadyExists.WithMsg(fmt.Sprintf("attēls ar nosaukumu '%s' jau eksistē", filename))
}

var ErrUnknownImageExt = srvcerror.New(
	"unknown_image_extension",
	"neatbalstīts attēla tips",
).SetHttpStatusCode(http.StatusBadRequest)

func errUnknownImageExt(mime string) srvcerror.E {
	return ErrUnknownImageExt.WithMsg(fmt.Sprintf("neatbalstīts attēla tips '%s'", mime))
}

var ErrImageDimensions = srvcerror.New(
	"image_dimensions",
	"attēla izmēru nevar nolasīt",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrImageSize = srvcerror.New(
	"image_inadequate_dimensions",
	"attēls ir pārāk mazs vai pārāk liels",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrInvalidTaskZip = srvcerror.New(
	"invalid_task_zip",
	"nederīgs TaskZip",
).SetHttpStatusCode(http.StatusBadRequest)

func errInvalidTaskZip(reason string) srvcerror.E {
	return ErrInvalidTaskZip.WithMsg(fmt.Sprintf("nederīgs TaskZip: %s", reason))
}

var ErrUnsupportedTaskZip = srvcerror.New(
	"unsupported_task_zip",
	"neatbalstīts TaskZip",
).SetHttpStatusCode(http.StatusBadRequest)

func errUnsupportedTaskZip(reason string) srvcerror.E {
	return ErrUnsupportedTaskZip.WithMsg(fmt.Sprintf("neatbalstīts TaskZip: %s", reason))
}

var ErrTaskAlreadyExists = srvcerror.New(
	"task_already_exists",
	"uzdevums ar šādu ID jau eksistē",
).SetHttpStatusCode(http.StatusConflict)

func errTaskAlreadyExists(taskId string) srvcerror.E {
	return ErrTaskAlreadyExists.WithMsg(fmt.Sprintf("uzdevums ar ID '%s' jau eksistē", taskId))
}

var ErrImageNotFound = srvcerror.New(
	"image_not_found",
	"attēls netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

func errImageNotFound(filename string) srvcerror.E {
	return ErrImageNotFound.WithMsg(fmt.Sprintf("attēls ar nosaukumu '%s' netika atrasts", filename))
}

var ErrIllustrationNotFound = srvcerror.New(
	"illustration_not_found",
	"uzdevumam nav ilustrācijas",
).SetHttpStatusCode(http.StatusNotFound)
