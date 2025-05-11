package srvc

import (
	"fmt"

	"github.com/programme-lv/backend/common/srvcerror"
)

const ErrCodeTaskNotFound = "task_not_found"

func NewErrorTaskNotFound(id string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeTaskNotFound,
		fmt.Sprintf("uzdevums '%s' netika atrasts", id),
	)
}

const ErrCodeImageAlreadyExists = "image_already_exists"

func NewErrorImageAlreadyExists(filename string) *srvcerror.Error {
	return srvcerror.New(
		ErrCodeImageAlreadyExists,
		fmt.Sprintf("attēls ar nosaukumu '%s' jau eksistē", filename),
	)
}
