package jsonresp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

type JsonResponse struct {
	Status  string `json:"status"` // "success" or "error"
	Data    any    `json:"data"`
	ErrCode string `json:"code,omitempty"`
	ErrMsg  string `json:"message,omitempty"`
}

func Success(w http.ResponseWriter, data any) error {
	resp := JsonResponse{
		Status: "success",
		Data:   data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(resp)
}

func Error(w http.ResponseWriter, errMsg string, statusCode int, errCode string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: errCode,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(resp)
}

func InternalError(w http.ResponseWriter) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  "internal server error; please contact the administrator",
		ErrCode: "undefined_internal_server_error",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	return json.NewEncoder(w).Encode(resp)
}

func BadRequest(w http.ResponseWriter, errMsg string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: "http_bad_request",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	return json.NewEncoder(w).Encode(resp)
}

func Forbidden(w http.ResponseWriter, errMsg string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: "http_forbidden",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	return json.NewEncoder(w).Encode(resp)
}

func Unauthorized(w http.ResponseWriter, errMsg string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: "http_unauthorized",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	return json.NewEncoder(w).Encode(resp)
}

func HandleSrvcError(logger *slog.Logger, w http.ResponseWriter, err error) {
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
		Error(w, srvcErr.Error(), srvcErr.HttpStatusCode(), srvcErr.ErrorCode())
		return
	} else {
		logger.Error("internal server error", "error", err)
		InternalError(w)
	}
}

// HandleErrorWithContext is a convenience function that extracts the logger from the context
func HandleErrorWithContext(ctx context.Context, w http.ResponseWriter, err error) {
	log := ctxlog.FromContext(ctx)
	HandleSrvcError(log, w, err)
}
