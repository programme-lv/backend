package httpfunc

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/programme-lv/backend/common/jsonresp"
)

type HandlerNoReq[R any] func(ctx context.Context) (response R, err jsonresp.HttpStatusCoder)

func NoReqJsonResp[R any](handler HandlerNoReq[R]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result, err := handler(ctx)
		if err != nil {
			jsonresp.WriteError(w, err)
			return
		}
		jsonresp.Success(w, result)
	}
}

type HandlerNoResp[Q any] func(ctx context.Context, request Q) (err jsonresp.HttpStatusCoder)

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
			jsonresp.WriteError(w, err)
			return
		}
		jsonresp.Success(w, struct{}{})
	}
}

type HandlerNoReqNoResp func(ctx context.Context) (err jsonresp.HttpStatusCoder)

func NoReqNoResp(handler HandlerNoReqNoResp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := handler(ctx); err != nil {
			jsonresp.WriteError(w, err)
			return
		}
		jsonresp.Success(w, struct{}{})
	}
}
