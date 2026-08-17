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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("write json success response", "error", err)
		return err
	}
	return nil
}

func WriteCustom(w http.ResponseWriter, errMsg string, statusCode int, errCode string) error {
	resp := JsonResponse{
		Status:  "error",
		ErrMsg:  errMsg,
		ErrCode: errCode,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("write json error response", "error", err)
		return err
	}
	return nil
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

// HandleSrvcError writes a service error as JSON.
// It does not log srvcerror values; the service already logged unexpected failures.
// logger is used only when err is not a srvcerror.E (a programming mistake at the HTTP boundary).
func HandleSrvcError(logger *slog.Logger, w http.ResponseWriter, err error) {
	if _, ok := err.(srvcerror.E); ok {
		WriteError(w, err)
		return
	}
	if logger != nil {
		logger.Error("internal server error", "error", err)
	}
	InternalError(w)
}

// HandleErrorWithContext is a convenience function that extracts the logger from the context
func HandleErrorWithContext(ctx context.Context, w http.ResponseWriter, err error) {
	log := ctxlog.FromContext(ctx)
	HandleSrvcError(log, w, err)
}
