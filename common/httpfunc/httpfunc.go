package httpfunc

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/programme-lv/backend/common/jsonresp"
)

type HandlerFuncImpl[Q any, R any] func(ctx context.Context, request Q) (response R, err error)

func Json[Q any, R any](handler HandlerFuncImpl[Q, R]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req Q
		t := reflect.TypeOf(req)
		if t.Size() > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonresp.BadRequest(w, err.Error())
				return
			}
		}
		result, err := handler(ctx, req)
		if err != nil {
			writeError(w, err)
			return
		}
		jsonresp.Success(w, result)
	}
}

type HandlerNoReq[R any] func(ctx context.Context) (response R, err error)

func NoReqJsonResp[R any](handler HandlerNoReq[R]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result, err := handler(ctx)
		if err != nil {
			writeError(w, err)
			return
		}
		jsonresp.Success(w, result)
	}
}

type HandlerNoResp[Q any] func(ctx context.Context, request Q) (err error)

func JsonReqNoResp[Q any](handler HandlerNoResp[Q]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req Q
		t := reflect.TypeOf(req)
		if t.Size() > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonresp.BadRequest(w, err.Error())
				return
			}
		}
		if err := handler(ctx, req); err != nil {
			writeError(w, err)
			return
		}
		jsonresp.Success(w, struct{}{})
	}
}

type HandlerNoReqNoResp func(ctx context.Context) (err error)

func NoReqNoResp(handler HandlerNoReqNoResp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := handler(ctx); err != nil {
			writeError(w, err)
			return
		}
		jsonresp.Success(w, struct{}{})
	}
}

func writeError(w http.ResponseWriter, err error) {
	type httpStatusCoder interface {
		Error() string
		HttpStatusCode() int
		ErrorCode() string
	}
	e, ok := err.(httpStatusCoder)
	if !ok {
		jsonresp.InternalError(w)
		return
	}
	jsonresp.Error(w, e.Error(), e.HttpStatusCode(), e.ErrorCode())
}
