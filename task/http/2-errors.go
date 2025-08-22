package http

import (
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/task/srvc"
)

func httpStatus(e srvcerror.Error) int {
	codeMap := map[string]int{
		srvc.ErrCodeImageAlreadyExists:        http.StatusConflict,
		srvc.ErrCodeTaskNotFound:              http.StatusNotFound,
		srvc.ErrCodeTaskAlreadyExists:         http.StatusConflict,
		srvc.ErrCodeInternalServerError:       http.StatusInternalServerError,
		srvc.ErrCodeImageFileExtFromMimeType:  http.StatusBadRequest,
		srvc.ErrCodeGetImageWidthAndHeight:    http.StatusBadRequest,
		srvc.ErrCodeImageInadequateDimensions: http.StatusBadRequest,
		srvc.ErrCodeFailedToGetTaskFromDb:     http.StatusInternalServerError,
	}

	if code, ok := codeMap[e.ErrorCode()]; ok {
		return code
	}

	return http.StatusInternalServerError
}
