package jsonresp

import (
	"context"
	"encoding/json"
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

func WriteCustom(w http.ResponseWriter, errMsg string, statusCode int, errCode string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: errCode,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(resp)
}

type HttpStatusCoder interface {
	Error() string
	HttpStatusCode() int
	ErrorCode() string
}

// assume error implements httpStatusCoder interface
func WriteError(w http.ResponseWriter, err error) {
	e, ok := err.(HttpStatusCoder)
	if !ok {
		debugMsg := "error does not implement httpStatusCoder interface"
		slog.Error(debugMsg, "error", err)

		InternalError(w)
	} else {
		WriteCustom(w,
			e.Error(),
			e.HttpStatusCode(),
			e.ErrorCode(),
		)
	}
}

// shorthand for jsonresp.FromError(w, ErrInternalServerError)
func InternalError(w http.ResponseWriter) {
	WriteError(w, ErrInternalServerError)
}

// shorthand for jsonresp.FromError(w, ErrHttpBadRequest.WithMsg(errMsg))
func BadRequest(w http.ResponseWriter, errMsg string) {
	WriteError(w, ErrHttpBadRequest.WithMsg(errMsg))
}

// shorthand for jsonresp.FromError(w, ErrHttpForbidden.WithMsg(errMsg))
func Forbidden(w http.ResponseWriter, errMsg string) {
	WriteError(w, ErrHttpForbidden.WithMsg(errMsg))
}

// shorthand for jsonresp.FromError(w, ErrHttpUnauthorized.WithMsg(errMsg))
func Unauthorized(w http.ResponseWriter, errMsg string) {
	WriteError(w, ErrHttpUnauthorized.WithMsg(errMsg))
}

// HandleSrvcError handles service errors and writes appropriate HTTP responses
func HandleSrvcError(logger *slog.Logger, w http.ResponseWriter, err error) {
	srvcErr, ok := err.(srvcerror.E)
	if ok {
		logger.Warn("service error", "error", err)
		// if srvcErr.HttpStatusCode() == http.StatusInternalServerError {
		// 	logger.Error("internal server error", "error", err)
		// }
		WriteCustom(w, srvcErr.Error(), srvcErr.HttpStatusCode(), srvcErr.ErrorCode())
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
