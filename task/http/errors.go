package http

import (
	"errors"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/task/srvc"
)

var (
	ErrBadRequest = errors.New("bad request")
)

func httpStatus(e error) int {
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

	if err, ok := e.(*srvcerror.Error); ok {
		if code, ok := codeMap[err.ErrorCode()]; ok {
			return code
		}
	}

	if errors.Is(e, ErrBadRequest) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}
