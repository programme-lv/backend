package srvc

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

const ErrCodeTaskNotFound = "task_not_found"

func NewErrorTaskNotFound(id string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeTaskNotFound,
		fmt.Sprintf("Uzdevums '%s' netika atrasts", id),
	).SetHttpStatusCode(http.StatusNotFound)
}
