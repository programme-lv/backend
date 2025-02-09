package httpjson

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/srvcerror"
)

type JsonResponse struct {
	Status  string `json:"status"` // "success" or "error"
	Data    any    `json:"data,omitempty"`
	ErrCode string `json:"code,omitempty"`
	ErrMsg  string `json:"message,omitempty"`
}

func WriteSuccessJson(w http.ResponseWriter, data any) {
	resp := JsonResponse{
		Status: "success",
		Data:   data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func WriteErrorJson(w http.ResponseWriter, errMsg string, statusCode int, errCode string) {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: errCode,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func writeInternalErrorJson(w http.ResponseWriter) {
	WriteErrorJson(w,
		http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError,
		"")
}

func HandleError(logger *slog.Logger, w http.ResponseWriter, err error) {
	srvcErr := &srvcerror.Error{}
	if errors.As(err, &srvcErr) {
		if srvcErr.DebugInfo() != nil {
			logger.Warn("service error", "error", err, "debug", srvcErr.DebugInfo())
		} else {
			logger.Warn("service error", "error", err)
		}
		if srvcErr.HttpStatusCode() == http.StatusInternalServerError {
			logger.Error("internal server error", "error", err)
		}
		WriteErrorJson(w, srvcErr.Error(), srvcErr.HttpStatusCode(), srvcErr.ErrorCode())
		return
	} else {
		logger.Error("internal server error", "error", err)
		writeInternalErrorJson(w)
	}
}
