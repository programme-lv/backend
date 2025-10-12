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

type HttpStatusCoder interface {
	Error() string
	HttpStatusCode() int
	ErrorCode() string
}

// assume error implements httpStatusCoder interface
func FromError(w http.ResponseWriter, err error) {
	e, ok := err.(HttpStatusCoder)
	if !ok {
		debugMsg := "error does not implement httpStatusCoder interface"
		slog.Error(debugMsg, "error", err)

		InternalError(w)
	} else {
		Error(w,
			e.Error(),
			e.HttpStatusCode(),
			e.ErrorCode(),
		)
	}
}

// shorthand for jsonresp.FromError(w, ErrInternalServerError)
func InternalError(w http.ResponseWriter) {
	FromError(w, ErrInternalServerError)
}

// shorthand for jsonresp.FromError(w, ErrHttpBadRequest.WithMsg(errMsg))
func BadRequest(w http.ResponseWriter, errMsg string) {
	FromError(w, ErrHttpBadRequest.WithMsg(errMsg))
}

// shorthand for jsonresp.FromError(w, ErrHttpForbidden.WithMsg(errMsg))
func Forbidden(w http.ResponseWriter, errMsg string) {
	FromError(w, ErrHttpForbidden.WithMsg(errMsg))
}

// shorthand for jsonresp.FromError(w, ErrHttpUnauthorized.WithMsg(errMsg))
func Unauthorized(w http.ResponseWriter, errMsg string) {
	FromError(w, ErrHttpUnauthorized.WithMsg(errMsg))
}

// deprecated
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
