package tasksrvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerr"
)

const ErrCodeTaskNotFound = "task_not_found"

func NewErrorTaskNotFound(id string) *srvcerr.Error {
	return srvcerr.New(
		ErrCodeTaskNotFound,
		fmt.Sprintf("Uzdevums '%s' netika atrasts", id),
	).SetHttpStatusCode(http.StatusNotFound)
}
