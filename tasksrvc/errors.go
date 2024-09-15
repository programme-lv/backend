package tasksrvc

import (
	"net/http"

	"github.com/programme-lv/backend/srvcerr"
)

const ErrCodeTaskNotFound = "task_not_found"

func newErrTaskNotFound() *srvcerr.Error {
	return srvcerr.New(
		ErrCodeTaskNotFound,
		"Uzdevums netika atrasts",
	).SetHttpStatusCode(http.StatusNotFound)
}
